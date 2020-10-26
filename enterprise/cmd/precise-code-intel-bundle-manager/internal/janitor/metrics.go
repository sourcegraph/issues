package janitor

import "github.com/prometheus/client_golang/prometheus"

type JanitorMetrics struct {
	UploadFilesRemoved   prometheus.Counter
	PartFilesRemoved     prometheus.Counter
	UploadRecordsRemoved prometheus.Counter
	DataRowsRemoved      prometheus.Counter
	Errors               prometheus.Counter
}

func NewJanitorMetrics(r prometheus.Registerer) JanitorMetrics {
	uploadFilesRemoved := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_bundle_manager_janitor_upload_files_removed_total",
		Help: "Total number of upload files removed (due to age)",
	})
	r.MustRegister(uploadFilesRemoved)

	partFilesRemoved := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_bundle_manager_janitor_upload_part_files_removed_total",
		Help: "Total number of upload part files removed (due to age)",
	})
	r.MustRegister(partFilesRemoved)

	uploadRecordsRemoved := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_bundle_manager_janitor_upload_records_removed_total",
		Help: "Total number of processed upload records removed",
	})
	r.MustRegister(uploadRecordsRemoved)

	dataRowsRemoved := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_bundle_manager_janitor_data_rows_removed_total",
		Help: "Total number of rows removed from the code intel database",
	})
	r.MustRegister(dataRowsRemoved)

	errors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "src_bundle_manager_janitor_errors_total",
		Help: "Total number of errors when running the janitor",
	})
	r.MustRegister(errors)

	return JanitorMetrics{
		UploadFilesRemoved:   uploadFilesRemoved,
		PartFilesRemoved:     partFilesRemoved,
		UploadRecordsRemoved: uploadRecordsRemoved,
		DataRowsRemoved:      dataRowsRemoved,
		Errors:               errors,
	}
}
