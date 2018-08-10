// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package push

import (
	"time"

	"github.com/go-kit/kit/log"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/storage"
)

type Pusher struct {
	app    storage.Appender
	logger log.Logger
}

func NewPusher(app storage.Appender, logger log.Logger) *Pusher {
	return &Pusher{
		app:    app,
		logger: logger,
	}
	return nil
}

func (p *Pusher) Push(data []byte) (total, added int, err error) {
	if len(data) == 0 {
		return 0, 0, nil
	}

	var (
		numOutOfOrder, numDuplicates, numOutOfBounds int

		now    = timestamp.FromTime(time.Now())
		parser = textparse.New(data)
	)

	level.Debug(p.logger).Log("msg", "Pushing data into Prometheus", "data", string(data))

loop:
	for parser.Next() {
		total++

		var labels labels.Labels

		buf, ts, val := parser.At()
		if ts == nil {
			ts = &now
		}
		parser.Metric(&labels)

		_, err := p.app.Add(labels, *ts, val)
		switch err {
		case nil:
			added++
		case storage.ErrOutOfOrderSample:
			level.Debug(p.logger).Log("msg", "Out of order sample", "series", string(buf))
			numOutOfOrder++
			continue
		case storage.ErrDuplicateSampleForTimestamp:
			level.Debug(p.logger).Log("msg", "Duplicate sample for timestamp", "series", string(buf))
			numDuplicates++
			continue
		case storage.ErrOutOfBounds:
			level.Debug(p.logger).Log("msg", "Out of bounds metric", "series", string(buf))
			numOutOfBounds++
			continue
		default:
			level.Debug(p.logger).Log("msg", "unexpected error", "series", string(buf), "err", err)
			break loop
		}
	}
	if err == nil {
		err = parser.Err()
	}
	if numOutOfOrder > 0 {
		level.Warn(p.logger).Log("msg", "Error on ingesting out-of-order samples", "num_dropped", numOutOfOrder)
	}
	if numDuplicates > 0 {
		level.Warn(p.logger).Log("msg", "Error on ingesting samples with different value but same timestamp", "num_dropped", numDuplicates)
	}
	if numOutOfBounds > 0 {
		level.Warn(p.logger).Log("msg", "Error on ingesting samples that are too old or are too far into the future", "num_dropped", numOutOfBounds)
	}
	if err != nil {
		p.app.Rollback()
		return total, added, err
	}
	if err := p.app.Commit(); err != nil {
		return total, added, err
	}
	return total, added, nil
}
