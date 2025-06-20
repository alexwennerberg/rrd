// Simple wrapper for rrdtool C library
package rrd

import (
	"fmt"
	"os"
	"math"
	"runtime"
	"strings"
	"time"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

/*
type cstring []byte

func newCstring(s string) cstring {
	cs := make(cstring, len(s)+1)
	copy(cs, s)
	return cs
}

func (cs cstring) p() unsafe.Pointer {
	if len(cs) == 0 {
		return nil
	}
	return unsafe.Pointer(&cs[0])
}

func (cs cstring) String() string {
	return string(cs[:len(cs)-1])
}
*/

func join(args []interface{}) string {
	sa := make([]string, len(args))
	for i, a := range args {
		var s string
		switch v := a.(type) {
		case time.Time:
			s = i64toa(v.Unix())
		default:
			s = fmt.Sprint(v)
		}
		sa[i] = s
	}
	return strings.Join(sa, ":")
}

type Creator struct {
	filename string
	start    time.Time
	step     uint
	noOverwrite int // bool
	source []string
	template string
	args     []string
}

// NewCreator returns new Creator object. You need to call Create to really
// create database file.
//	filename - name of database file
//	start    - don't accept any data timed before or at time specified
//	step     - base interval in seconds with which data will be fed into RRD
func NewCreator(filename string, start time.Time, step uint) *Creator {
	return &Creator{
		filename: filename,
		start:    start,
		step:     step,
	}
}

// DS formats a DS argument and appends it to the list of arguments to be
// passed to rrdcreate(). Each element of args is formatted with fmt.Sprint().
// Please see the rrdcreate(1) manual page for in-depth documentation.
func (c *Creator) DS(name, compute string, args ...interface{}) {
	c.args = append(c.args, "DS:"+name+":"+compute+":"+join(args))
}

// RRA formats an RRA argument and appends it to the list of arguments to be
// passed to rrdcreate(). Each element of args is formatted with fmt.Sprint().
// Please see the rrdcreate(1) manual page for in-depth documentation.
func (c *Creator) RRA(cf string, args ...interface{}) {
	c.args = append(c.args, "RRA:"+cf+":"+join(args))
}

// SetSource sets an existing source file(s) to read data from
// Please see the rrdcreate(1) manual page for in-depth documentation.
func (c *Creator) SetSource(s ...string) {
	c.source = s
}

func (c *Creator) SetTemplate(t ...string) {
	c.template = strings.Join(t, ":")
}

func (c *Creator) SetNoOverwite() {
	c.noOverwrite = 1
}

// Create creates new database file. If overwrite is true it overwrites
// database file if exists. If overwrite is false it returns error if file
// exists (you can use os.IsExist function to check this case).
// Deprecated in favor of SetNoOverwrite(), which uses the native RRD flag
func (c *Creator) Create(overwrite bool) error {
	if !overwrite {
		f, err := os.OpenFile(
			c.filename,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
			0666,
		)
		if err != nil {
			return err
		}
		f.Close()
	}
	return c.create()
}

// Use cstring and unsafe.Pointer to avoid allocations for C calls

type Updater struct {
	filename *cstring
	template *cstring

	args []*cstring
}

func NewUpdater(filename string) *Updater {
	u := &Updater{filename: newCstring(filename)}
	runtime.SetFinalizer(u, cfree)
	return u
}

func cfree(u *Updater) {
	u.filename.Free()
	u.template.Free()
	for _, a := range u.args {
		a.Free()
	}
}

func (u *Updater) SetTemplate(dsName ...string) {
	u.template.Free()
	u.template = newCstring(strings.Join(dsName, ":"))
}

// Cache chaches data for later save using Update(). Use it to avoid
// open/read/write/close for every update.
func (u *Updater) Cache(args ...interface{}) {
	u.args = append(u.args, newCstring(join(args)))
}

// Update saves data in RRDB.
// Without args Update saves all subsequent updates buffered by Cache method.
// If you specify args it saves them immediately.
func (u *Updater) Update(args ...interface{}) error {
	if len(args) != 0 {
		cs := newCstring(join(args))
		err := u.update([]*cstring{cs})
		cs.Free()
		return err
	} else if len(u.args) != 0 {
		err := u.update(u.args)
		for _, a := range u.args {
			a.Free()
		}
		u.args = nil
		return err
	}
	return nil
}

type GraphInfo struct {
	Print         []string
	Width, Height uint
	Ymin, Ymax    float64
}

type Grapher struct {
	title           string
	vlabel          string
	width, height   uint
	borderWidth     uint
	upperLimit      float64
	lowerLimit      float64
	rigid           bool
	altAutoscale    bool
	altAutoscaleMin bool
	altAutoscaleMax bool
	noGridFit       bool

	logarithmic   bool
	unitsExponent int
	unitsLength   uint

	rightAxisScale float64
	rightAxisShift float64
	rightAxisLabel string

	noLegend bool

	lazy bool

	colors map[string]string

	slopeMode bool

	watermark   string
	base        uint
	imageFormat string
	interlaced  bool

	daemon string

	args []string
}

const (
	maxUint  = ^uint(0)
	maxInt   = int(maxUint >> 1)
	minInt   = -maxInt - 1
	defWidth = 2
)

func NewGrapher() *Grapher {
	return &Grapher{
		upperLimit:    -math.MaxFloat64,
		lowerLimit:    math.MaxFloat64,
		unitsExponent: minInt,
		borderWidth:   defWidth,
		colors:        make(map[string]string),
	}
}

func (g *Grapher) SetTitle(title string) {
	g.title = title
}

func (g *Grapher) SetVLabel(vlabel string) {
	g.vlabel = vlabel
}

func (g *Grapher) SetSize(width, height uint) {
	g.width = width
	g.height = height
}

func (g *Grapher) SetBorder(width uint) {
	g.borderWidth = width
}

func (g *Grapher) SetLowerLimit(limit float64) {
	g.lowerLimit = limit
}

func (g *Grapher) SetUpperLimit(limit float64) {
	g.upperLimit = limit
}

func (g *Grapher) SetRigid() {
	g.rigid = true
}

func (g *Grapher) SetAltAutoscale() {
	g.altAutoscale = true
}

func (g *Grapher) SetAltAutoscaleMin() {
	g.altAutoscaleMin = true
}

func (g *Grapher) SetAltAutoscaleMax() {

	g.altAutoscaleMax = true
}

func (g *Grapher) SetNoGridFit() {
	g.noGridFit = true
}

func (g *Grapher) SetLogarithmic() {
	g.logarithmic = true
}

func (g *Grapher) SetUnitsExponent(e int) {
	g.unitsExponent = e
}

func (g *Grapher) SetUnitsLength(l uint) {
	g.unitsLength = l
}

func (g *Grapher) SetRightAxis(scale, shift float64) {
	g.rightAxisScale = scale
	g.rightAxisShift = shift
}

func (g *Grapher) SetRightAxisLabel(label string) {
	g.rightAxisLabel = label
}

func (g *Grapher) SetNoLegend() {
	g.noLegend = true
}

func (g *Grapher) SetLazy() {
	g.lazy = true
}

func (g *Grapher) SetColor(colortag, color string) {
	g.colors[colortag] = color
}

func (g *Grapher) SetSlopeMode() {
	g.slopeMode = true
}

func (g *Grapher) SetImageFormat(format string) {
	g.imageFormat = format
}

func (g *Grapher) SetInterlaced() {
	g.interlaced = true
}

func (g *Grapher) SetBase(base uint) {
	g.base = base
}

func (g *Grapher) SetWatermark(watermark string) {
	g.watermark = watermark
}

func (g *Grapher) SetDaemon(daemon string) {
	g.daemon = daemon
}

func (g *Grapher) AddOptions(options ...string) {
	g.args = append(g.args, options...)
}

func (g *Grapher) push(cmd string, options []string) {
	if len(options) > 0 {
		cmd += ":" + strings.Join(options, ":")
	}
	g.args = append(g.args, cmd)
}

func (g *Grapher) Def(vname, rrdfile, dsname, cf string, options ...string) {
	g.push(
		fmt.Sprintf("DEF:%s=%s:%s:%s", vname, rrdfile, dsname, cf),
		options,
	)
}

func (g *Grapher) VDef(vname, rpn string) {
	g.push("VDEF:"+vname+"="+rpn, nil)
}

func (g *Grapher) CDef(vname, rpn string) {
	g.push("CDEF:"+vname+"="+rpn, nil)
}

func (g *Grapher) Print(vname, format string) {
	g.push("PRINT:"+vname+":"+format, nil)
}

func (g *Grapher) PrintT(vname, format string) {
	g.push("PRINT:"+vname+":"+format+":strftime", nil)
}
func (g *Grapher) GPrint(vname, format string) {
	g.push("GPRINT:"+vname+":"+format, nil)
}

func (g *Grapher) GPrintT(vname, format string) {
	g.push("GPRINT:"+vname+":"+format+":strftime", nil)
}

func (g *Grapher) Comment(s string) {
	g.push("COMMENT:"+s, nil)
}

func (g *Grapher) VRule(t interface{}, color string, options ...string) {
	if v, ok := t.(time.Time); ok {
		t = v.Unix()
	}
	vr := fmt.Sprintf("VRULE:%v#%s", t, color)
	g.push(vr, options)
}

func (g *Grapher) HRule(value, color string, options ...string) {
	hr := "HRULE:" + value + "#" + color
	g.push(hr, options)
}

func (g *Grapher) Line(width float32, value, color string, options ...string) {
	line := fmt.Sprintf("LINE%f:%s", width, value)
	if color != "" {
		line += "#" + color
	}
	g.push(line, options)
}

func (g *Grapher) Area(value, color string, options ...string) {
	area := "AREA:" + value
	if color != "" {
		area += "#" + color
	}
	g.push(area, options)
}

func (g *Grapher) Tick(vname, color string, options ...string) {
	tick := "TICK:" + vname
	if color != "" {
		tick += "#" + color
	}
	g.push(tick, options)
}

func (g *Grapher) Shift(vname string, offset interface{}) {
	if v, ok := offset.(time.Duration); ok {
		offset = int64((v + time.Second/2) / time.Second)
	}
	shift := fmt.Sprintf("SHIFT:%s:%v", vname, offset)
	g.push(shift, nil)
}

func (g *Grapher) TextAlign(align string) {
	g.push("TEXTALIGN:"+align, nil)
}

// Graph returns GraphInfo and image as []byte or error
func (g *Grapher) Graph(start, end time.Time) (GraphInfo, []byte, error) {
	return g.graph("-", start, end)
}

// SaveGraph saves image to file and returns GraphInfo or error
func (g *Grapher) SaveGraph(filename string, start, end time.Time) (GraphInfo, error) {
	gi, _, err := g.graph(filename, start, end)
	return gi, err
}

type FetchResult struct {
	Filename string
	Cf       string
	Start    time.Time
	End      time.Time
	Step     time.Duration
	DsNames  []string
	RowCnt   int
	values   []float64
}

func (r *FetchResult) ValueAt(dsIndex, rowIndex int) float64 {
	return r.values[len(r.DsNames)*rowIndex+dsIndex]
}

type Exporter struct {
	maxRows uint

	daemon string

	args []string
}

func NewExporter() *Exporter {
	return &Exporter{}
}

func (e *Exporter) SetMaxRows(maxRows uint) {
	e.maxRows = maxRows
}

func (e *Exporter) push(cmd string, options []string) {
	if len(options) > 0 {
		cmd += ":" + strings.Join(options, ":")
	}
	e.args = append(e.args, cmd)
}

func (e *Exporter) Def(vname, rrdfile, dsname, cf string, options ...string) {
	e.push(
		fmt.Sprintf("DEF:%s=%s:%s:%s", vname, rrdfile, dsname, cf),
		options,
	)
}

func (e *Exporter) CDef(vname, rpn string) {
	e.push("CDEF:"+vname+"="+rpn, nil)
}

func (e *Exporter) XportDef(vname, label string) {
	e.push("XPORT:"+vname+":"+label, nil)
}

func (e *Exporter) Xport(start, end time.Time, step time.Duration) (XportResult, error) {
	return e.xport(start, end, step)
}

func (e *Exporter) SetDaemon(daemon string) {
	e.daemon = daemon
}

type XportResult struct {
	Start   time.Time
	End     time.Time
	Step    time.Duration
	Legends []string
	RowCnt  int
	values  []float64
}

func (r *XportResult) ValueAt(legendIndex, rowIndex int) float64 {
	return r.values[len(r.Legends)*rowIndex+legendIndex]
}
