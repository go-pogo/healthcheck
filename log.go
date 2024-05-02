// Copyright (c) 2023, Roel Schut. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package healthcheck

import "log"

type Logger interface {
	Notifier
	StatusChecked(name string, stat Status)
}

var _ Logger = (*logger)(nil)

type logger struct {
	*log.Logger
}

func (l *logger) StatusChanged(status, oldStatus Status) {
	l.Logger.Println("status changed from " + oldStatus.String() + " to " + status.String())
}

func (l *logger) StatusChecked(name string, stat Status) {
	l.Logger.Printf("status for %s is %s\n", name, stat)
}
