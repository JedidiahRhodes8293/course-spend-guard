package coursebudget

import (
	"context"
	"fmt"
)

type LessonRequest struct {
	CourseID       string  `json:"course_id"`
	LearnerID      string  `json:"learner_id"`
	Deadline       string  `json:"deadline"`
	HardCapUSD     float64 `json:"hard_cap_usd"`
	AlertUSD       float64 `json:"alert_threshold_usd"`
	EducatorPrompt string  `json:"educator_prompt"`
}

type DeliveryResult struct {
	CourseID string `json:"course_id"`
	Deadline string `json:"deadline"`
	Report   any    `json:"report"`
	Usage    any    `json:"usage_timeseries"`
}

type platform interface {
	SetCourseBudget(context.Context, float64, string, float64) error
	UsageTimeseries(context.Context, any) error
	ChatCompletion(context.Context, string, any) error
}

func DeliverLesson(ctx context.Context, api platform, input LessonRequest) (DeliveryResult, error) {
	if input.CourseID == "" || input.LearnerID == "" || input.Deadline == "" || input.HardCapUSD <= 0 || input.EducatorPrompt == "" {
		return DeliveryResult{}, fmt.Errorf("course_id, learner_id, deadline, hard_cap_usd, and educator_prompt are required")
	}
	if input.AlertUSD <= 0 || input.AlertUSD > input.HardCapUSD {
		return DeliveryResult{}, fmt.Errorf("alert_threshold_usd must be greater than zero and no greater than hard_cap_usd")
	}
	if err := api.SetCourseBudget(ctx, input.HardCapUSD, "monthly", input.AlertUSD); err != nil {
		return DeliveryResult{}, err
	}
	var usage any
	if err := api.UsageTimeseries(ctx, &usage); err != nil {
		return DeliveryResult{}, err
	}
	prompt := fmt.Sprintf("Course %s, learner %s, deadline %s. %s", input.CourseID, input.LearnerID, input.Deadline, input.EducatorPrompt)
	var report any
	if err := api.ChatCompletion(ctx, prompt, &report); err != nil {
		return DeliveryResult{}, err
	}
	return DeliveryResult{CourseID: input.CourseID, Deadline: input.Deadline, Report: report, Usage: usage}, nil
}
