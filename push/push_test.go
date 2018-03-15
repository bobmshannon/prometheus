package push

import (
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushSamples(t *testing.T) {
	tests := []struct {
		name            string
		input           []byte
		expectedAdded   int
		expectedTotal   int
		expectedSamples []sample
		invalidSamples  bool
	}{
		{
			"Empty payload",
			[]byte{},
			0,
			0,
			[]sample{},
			false,
		},
		{
			"Payload missing trailing newline",
			[]byte(`# HELP CONTOUR_BACKEND_CACHE_relay_metrics_completed_1m Telegraf collected metric
# TYPE CONTOUR_BACKEND_CACHE_relay_metrics_completed_1m untyped
CONTOUR_BACKEND_CACHE_relay_metrics_completed_1m{label="value"} 0 16204295550011`),
			1,
			1,
			[]sample{
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "CONTOUR_BACKEND_CACHE_relay_metrics_completed_1m",
						},
						labels.Label{
							Name:  "label",
							Value: "value",
						},
					},
					16204295550011,
					0,
				},
			},
			false,
		},
		{
			"Push valid samples",
			[]byte(`# HELP CONTOUR_BACKEND_CACHE_relay_metrics_completed_1m Telegraf collected metric
# TYPE CONTOUR_BACKEND_CACHE_relay_metrics_completed_1m untyped
CONTOUR_BACKEND_CACHE_relay_metrics_completed_1m{label="value"} 0 16204295550011
# HELP CONTOUR_BACKEND_CACHE_relay_metrics_completed_count Telegraf collected metric
# TYPE CONTOUR_BACKEND_CACHE_relay_metrics_completed_count untyped
CONTOUR_BACKEND_CACHE_relay_metrics_completed_count{label="value"} 0 16204295550011
# HELP CONTOUR_BACKEND_CACHE_relay_metrics_duration_1m Telegraf collected metric
# TYPE CONTOUR_BACKEND_CACHE_relay_metrics_duration_1m untyped
CONTOUR_BACKEND_CACHE_relay_metrics_duration_1m{label="value"} 0 16204295550011
# HELP CONTOUR_BACKEND_MULTIPLEXER_relay_metrics_running_count Telegraf collected metric
# TYPE CONTOUR_BACKEND_MULTIPLEXER_relay_metrics_running_count counter
CONTOUR_BACKEND_MULTIPLEXER_relay_metrics_running_count{label="value"} 0 16204295550011
`),
			4,
			4,
			[]sample{
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "CONTOUR_BACKEND_CACHE_relay_metrics_completed_1m",
						},
						labels.Label{
							Name:  "label",
							Value: "value",
						},
					},
					16204295550011,
					0,
				},
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "CONTOUR_BACKEND_CACHE_relay_metrics_completed_count",
						},
						labels.Label{
							Name:  "label",
							Value: "value",
						},
					},
					16204295550011,
					0,
				},
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "CONTOUR_BACKEND_CACHE_relay_metrics_duration_1m",
						},
						labels.Label{
							Name:  "label",
							Value: "value",
						},
					},
					16204295550011,
					0,
				},
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "CONTOUR_BACKEND_MULTIPLEXER_relay_metrics_running_count",
						},
						labels.Label{
							Name:  "label",
							Value: "value",
						},
					},
					16204295550011,
					0,
				},
			},
			false,
		},
		{
			"Push mix of valid and invalid samples",
			[]byte(`# HELP hostmetric_cpu_usage_iowait Telegraf collected metric
# TYPE hostmetric_cpu_usage_iowait gauge
hostmetric_cpu_usage_iowait{cpu="cpu-total"} 0.06018054162487866 15204295550000
hostmetric_cpu_usage_iowait{cpu="cpu0"} 0.35495321071313496 15204295550000
hostmetric_cpu_usage_iowait{cpu="cpu1"} 0.03226847370119539 15204295550000
hostmetric_cpu_usage_iowait{cpu="cpu2"} 0.032237266279825004 15204295550000
hostmetric_cpu_usage_iowait{cpu="cpu3"} 0.03221649484535923 15204295550000
hostmetric_cpu_usage_iowait{cpu="cpu4
`),
			5,
			5,
			[]sample{
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "hostmetric_cpu_usage_iowait",
						},
						labels.Label{
							Name:  "cpu",
							Value: "cpu-total",
						},
					},
					15204295550000,
					0.06018054162487866,
				},
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "hostmetric_cpu_usage_iowait",
						},
						labels.Label{
							Name:  "cpu",
							Value: "cpu0",
						},
					},
					15204295550000,
					0.35495321071313496,
				},
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "hostmetric_cpu_usage_iowait",
						},
						labels.Label{
							Name:  "cpu",
							Value: "cpu1",
						},
					},
					15204295550000,
					0.03226847370119539,
				},
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "hostmetric_cpu_usage_iowait",
						},
						labels.Label{
							Name:  "cpu",
							Value: "cpu2",
						},
					},
					15204295550000,
					0.032237266279825004,
				},
				{
					labels.Labels{
						labels.Label{
							Name:  labels.MetricName,
							Value: "hostmetric_cpu_usage_iowait",
						},
						labels.Label{
							Name:  "cpu",
							Value: "cpu3",
						},
					},
					15204295550000,
					0.03221649484535923,
				},
			},
			true,
		},
	}

	for _, test := range tests {
		app := &collectResultAppender{}
		logger := log.NewNopLogger()
		pusher := NewPusher(app, logger)

		actualTotal, actualAdded, err := pusher.Push(test.input)
		if test.invalidSamples {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		assert.Equal(t, test.expectedAdded, actualAdded)
		assert.Equal(t, test.expectedTotal, actualTotal)
		for idx, sample := range app.result {
			assert.Equal(t, test.expectedSamples[idx], sample)
		}
	}
}
