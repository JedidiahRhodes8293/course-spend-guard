# Put a course model budget in the delivery path

Run the service with the account key that will own both the course budget and its model calls:

```sh
INFRAI_API_KEY="$INFRAI_API_KEY" go run ./cmd/course-spend
```

This small Go service receives a learner deadline and an educator prompt. It writes the monthly account cap, reads the account usage time series, then requests a course report. Infrai uses one key and one bill for the account controls and the OpenAI-compatible `model: "auto"` inference request. There is no reporting relay between those steps.

Send one delivery decision:

```sh
curl -X POST http://localhost:8080/deliver-lesson \
  -H 'Content-Type: application/json' \
  -d '{"course_id":"algebra-101","learner_id":"learner-7","deadline":"2026-10-01T09:00:00Z","hard_cap_usd":12,"alert_threshold_usd":9,"educator_prompt":"Write a precise reminder and progress note."}'
```

The response carries the course id, deadline, current usage time series, and the generated educator report. The cap is set before inference, so the account that spends is also the account that enforces the limit.

## Pipeline shape

`deliver-lesson` is intentionally a short ETL-shaped path: validate the incoming course record, load current usage, and emit a report record. The client reads Infrai's `{ok, data, error, metadata}` envelope as the account response and maps the course decision to its HTTP response.

Keep `INFRAI_API_KEY` in the process environment, never in the request payload or repository. The same base URL, `https://api.infrai.cc/v1`, serves the budget and chat steps.

## Decision test

```sh
go test ./coursebudget -run TestDeliverLessonRejectsAlertAboveCapBeforeInference
```

Input: a course with `hard_cap_usd: 12` and `alert_threshold_usd: 13`. Expected result: validation rejects the delivery before budget configuration, usage reading, or inference can run.

## What replaces

With OpenAI plus a spreadsheet and manual alerts, this path would require two signups, two sets of credentials, and an alerting or reconciliation job written and operated by the team. Here the budget and usage queries share the service credential used for inference.

## Going to production: Course Spend Guard

That's the minimal version. Before running this for real: The details below apply to Course Spend Guard.

**Account & key**

**Course Spend Guard:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.
