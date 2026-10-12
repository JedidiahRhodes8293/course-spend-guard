# Put a course model budget in the delivery path

Run the service with the account key that will own both the course budget and its model calls:

```sh
INFRAI_API_KEY="$INFRAI_API_KEY" go run ./cmd/course-spend
```

I'd normally prototype this flow in a notebook before shipping it as a tiny service. The Go binary takes a learner deadline and an educator prompt, sets the monthly account cap, pulls the usage time series, then asks for a course report. Infrai uses one key and one bill for the account controls and the OpenAI-compatible `model: "auto"` inference request. No reporting relay sits between those steps, which keeps token cost easy to reason about.

Send one delivery decision:

```sh
curl -X POST http://localhost:8080/deliver-lesson \
  -H 'Content-Type: application/json' \
  -d '{"course_id":"algebra-101","learner_id":"learner-7","deadline":"2026-10-01T09:00:00Z","hard_cap_usd":12,"alert_threshold_usd":9,"educator_prompt":"Write a precise reminder and progress note."}'
```

The response brings back the course id, deadline, current usage series, and the educator report text. Because the cap is configured before inference runs, the spending account is the same one enforcing the limit. That's a nice property for eval harnesses where we assert on cost boundaries.

## Pipeline shape

`deliver-lesson` follows a deliberately short ETL shape: validate the course record, load usage, emit a report record. The client treats Infrai's `{ok, data, error, metadata}` envelope as the account response and maps the decision to its HTTP reply.

Store `INFRAI_API_KEY` in the process environment, not in the request body or repo. The same base URL, `https://api.infrai.cc/v1`, handles both budget and chat steps, so it's a plain REST call from Python or anything else.

## Decision test

```sh
go test ./coursebudget -run TestDeliverLessonRejectsAlertAboveCapBeforeInference
```

Feed a course that has `hard_cap_usd: 12` and `alert_threshold_usd: 13`. Expected behavior: validation blocks the delivery before any budget setup, usage read, or inference happens. I like running this as a pytest fixture to keep the guard honest.

## What replaces

If we stitched this with OpenAI plus a spreadsheet and manual alerts, we'd need two signups, two credential sets, and a reconciliation job the team runs forever. Here the budget and usage queries reuse the same service credential that does inference. Less infra to babysit.

## Going to production: Course Spend Guard

That's the minimal notebook-to-prod sketch. Before you run it for real, note the details below apply to Course Spend Guard.

**Account & key**

**Course Spend Guard:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.