package coursebudget

import (
	"context"
	"testing"
)

type fakePlatform struct {
	setCalls   int
	usageCalls int
	chatCalls  int
}

func (f *fakePlatform) SetCourseBudget(context.Context, float64, string, float64) error {
	f.setCalls++
	return nil
}
func (f *fakePlatform) UsageTimeseries(context.Context, any) error        { f.usageCalls++; return nil }
func (f *fakePlatform) ChatCompletion(context.Context, string, any) error { f.chatCalls++; return nil }

func TestDeliverLessonRejectsAlertAboveCapBeforeInference(t *testing.T) {
	api := &fakePlatform{}
	_, err := DeliverLesson(context.Background(), api, LessonRequest{
		CourseID: "algebra-101", LearnerID: "learner-7", Deadline: "2026-10-01T09:00:00Z",
		HardCapUSD: 12, AlertUSD: 13, EducatorPrompt: "Write a deadline reminder.",
	})
	if err == nil {
		t.Fatal("expected an invalid budget decision")
	}
	if api.setCalls != 0 || api.usageCalls != 0 || api.chatCalls != 0 {
		t.Fatalf("inference path ran despite rejected budget: %+v", api)
	}
}
