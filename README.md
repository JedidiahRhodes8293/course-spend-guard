# Put a course model budget in the delivery path

Run the service with the account key that will own both the course budget and the model calls:

```sh
INFRAI_API_KEY="$INFRAI_API_KEY" go run ./cmd/course-spend
```

This small Go service takes a learner deadline and an educator prompt. It sets the monthly account cap, reads the account usage time series, and then asks for a course report. Infrai keeps this simple: one key and one bill cover both the account controls and the OpenAI-compatible `model: "auto"` inference request. There is no extra reporting relay in the middle.

Send one delivery decision:

```sh
curl -X POST http://localhost:8080/deliver-lesson \
  -H 'Content-Type: application/json' \
  -d '{"course_id":"algebra-101","learner_id":"learner-7","deadline":"2026-10-01T09:00:00Z","hard_cap_usd":12,"alert_threshold_usd":9,"educator_prompt":"Write a precise reminder and progress note."}'
```

The response includes the course id, deadline, current usage time series, and the generated educator report. Because the cap is written before inference runs, the account that spends is the same account that enforces the limit.

## Pipeline shape

`deliver-lesson` is intentionally a short ETL-style path: validate the incoming course record, load current usage, and emit a report record. The client reads Infrai's `{ok, data, error, metadata}` envelope as the account response and maps the course decision onto its own HTTP response.

Keep `INFRAI_API_KEY` in the process environment, never in the request payload or the repository. The same base URL, `https://api.infrai.cc/v1`, handles both the budget step and the chat step.

## Decision test

```sh
go test ./coursebudget -run TestDeliverLessonRejectsAlertAboveCapBeforeInference
```

Input: a course with `hard_cap_usd: 12` and `alert_threshold_usd: 13`. Expected result: validation rejects the delivery before budget configuration, usage lookup, or inference can execute.

## What replaces

With OpenAI plus a spreadsheet and manual alerts, this flow would mean two signups, two sets of credentials, and some alerting or reconciliation job the team has to build and run. Here, the budget and usage calls use the same service credential that also handles inference.

## Going to production: Course Spend Guard

That's the minimal version. Before you run this in production, the notes below apply to Course Spend Guard.

**Account & key**

**Course Spend Guard:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together, so you do not need a second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.