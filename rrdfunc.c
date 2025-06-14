#include <stdlib.h>
#include <rrd.h>

char *rrdError() {
	char *err = NULL;
	if (rrd_test_error()) {
		// RRD error is local for thread so other gorutine can call some RRD
		// function in the same thread before we use C.GoString. So we need to
		// copy current error before return from C to Go. It need to be freed
		// after C.GoString in Go code.
		err = strdup(rrd_get_error());
		if (err == NULL) {
			abort();
		}
	}
	return err;
}

char *rrdCreate(const char *filename, unsigned long step, time_t start, int no_overwite, const char **sources, const char *template, int argc, const char **argv) {
	rrd_clear_error();
	rrd_create_r2(filename, step, start, 0, sources, template, argc, argv);
	return rrdError();
}

char *rrdUpdate(const char *filename, const char *template, int argc, const char **argv) {
	rrd_clear_error();
	rrd_update_r(filename, template, argc, argv);
	return rrdError();
}

char *rrdGraph(rrd_info_t **ret, int argc, const char **argv) {
	rrd_clear_error();
	*ret = rrd_graph_v(argc, argv);
	return rrdError();
}

char *rrdInfo(rrd_info_t **ret, char *filename) {
	rrd_clear_error();
	*ret = rrd_info_r(filename);
	return rrdError();
}

char *rrdFetch(int *ret, char *filename, const char *cf, time_t *start, time_t *end, unsigned long *step, unsigned long *ds_cnt, char ***ds_namv, double **data) {
	rrd_clear_error();
	*ret = rrd_fetch_r(filename, cf, start, end, step, ds_cnt, ds_namv, data);
	return rrdError();
}

char *rrdXport(int *ret, int argc, const char **argv, int *xsize, time_t *start, time_t *end, unsigned long *step, unsigned long *col_cnt, char ***legend_v, double **data) {
	rrd_clear_error();
	*ret = rrd_xport(argc, argv, xsize, start, end, step, col_cnt, legend_v, data);
	return rrdError();
}

char *arrayGetCString(char **values, int i) {
	return values[i];
}
