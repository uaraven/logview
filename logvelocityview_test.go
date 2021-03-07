package logview

import (
	"github.com/gdamore/tcell/v2"
	"testing"
	"time"
)

func TestLogVelocityView_AutoScale(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	screen.SetSize(10, 10)
	velocity := NewLogVelocityView(time.Minute)
	velocity.SetRect(0, 0, 10, 10)
	velocity.Draw(screen)
	start := time.Date(2021, 03, 01, 10, 0, 0, 0, time.Local)
	end := start.Add(20 * time.Minute)
	velocity.AutoScale(start, end)
	if velocity.bucketWidth != 120 {
		t.Errorf("Should have 2 minute bucket size, but got %d", velocity.bucketWidth)
	}
	end = start.Add(25 * time.Minute)
	velocity.AutoScale(start, end)
	if velocity.bucketWidth != 120 {
		t.Errorf("Should have 2 minute bucket size, but got %d", velocity.bucketWidth)
	}
	end = start.Add(55 * time.Minute)
	velocity.AutoScale(start, end)
	if velocity.bucketWidth != 300 {
		t.Errorf("Should have 5 minute bucket size, but got %d", velocity.bucketWidth)
	}
}

func TestLogVelocityView_Scale(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	screen.SetSize(10, 10)
	velocity := NewLogVelocityView(time.Minute)
	velocity.SetRect(0, 0, 10, 10)
	velocity.Draw(screen)
	velocity.ScaleFor(20 * time.Minute)
	if velocity.bucketWidth != 120 {
		t.Errorf("Should have 2 minute bucket size, but got %d", velocity.bucketWidth)
	}
	velocity.ScaleFor(25 * time.Minute)
	if velocity.bucketWidth != 120 {
		t.Errorf("Should have 2 minute bucket size, but got %d", velocity.bucketWidth)
	}
	velocity.ScaleFor(55 * time.Minute)
	if velocity.bucketWidth != 300 {
		t.Errorf("Should have 5 minute bucket size, but got %d", velocity.bucketWidth)
	}
}
