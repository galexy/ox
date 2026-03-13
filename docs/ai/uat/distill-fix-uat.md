<!-- doc-audience: ai -->
# UAT: ox distill Logical Fixes (#211)

## Prerequisites
- A SageOx-initialized repo with team context
- At least one AI coworker CLI (claude) available
- Observations recorded across multiple days (or manually created test fixtures)

## Scenario 1: Multi-Day Catch-Up

**Setup:** Don't run `ox distill` for 3+ days while observations accumulate.

**Steps:**
1. Record observations on Day 1: `ox memory put "Day 1 observation A"`, `ox memory put "Day 1 observation B"`
2. Wait (or manually create observation files dated Day 2 and Day 3)
3. Run `ox distill --dry-run`

**Expected:**
- Output lists 3 separate daily distillations, one per day
- Each day shows only the observation count for that day
- No single day contains observations from other days

4. Run `ox distill`

**Expected:**
- 3 daily files created: `memory/daily/YYYY-MM-DD-{uuid7}.md` for each day
- Each file's content references only that day's observations
- `distill-state-v2.json` shows `last_daily` = processing timestamp (RFC3339)

## Scenario 2: Intra-Day Multiple Runs

**Setup:** Run `ox distill` twice in the same day with new observations between runs.

**Steps:**
1. Record: `ox memory put "Morning observation"`
2. Run `ox distill`
3. Note the output filename (e.g., `memory/daily/2026-03-12-{uuid7-A}.md`)
4. Record: `ox memory put "Afternoon observation"`
5. Run `ox distill` again

**Expected:**
- Second run creates a NEW file: `memory/daily/2026-03-12-{uuid7-B}.md`
- First file `{uuid7-A}` is untouched
- Both files exist in `memory/daily/`
- First file contains "Morning observation" summary
- Second file contains "Afternoon observation" summary

## Scenario 3: Fresh Clone Recovery

**Setup:** Clone the repo to a new directory (no `distill-state-v2.json`), but daily files already exist in team context.

**Steps:**
1. In original repo, run `ox distill` to create daily files through today
2. Delete `{projectRoot}/.sageox/distill-state-v2.json`
3. Record a new observation: `ox memory put "Post-delete observation"`
4. Run `ox distill --dry-run`

**Expected:**
- Does NOT re-process all historical observations
- Only shows the new observation for today
- Output indicates high-water mark was inferred from existing daily files

5. Run `ox distill`

**Expected:**
- Only one new daily file created (for today, with the new observation)
- Existing daily files untouched

## Scenario 4: Multiple Clones

**Setup:** Two clones of the same repo, both pointed at the same team context.

**Steps:**
1. In Clone A: `ox memory put "Clone A observation"` then `ox distill`
2. In Clone B: `ox memory put "Clone B observation"` then `ox distill`

**Expected:**
- Both clones produce daily files for the same date
- Files have different UUID7 suffixes — no overwrite
- Both files exist in team context `memory/daily/`
- Some duplicate content across files is acceptable

## Scenario 5: Weekly Catch-Up Across Multiple Weeks

**Setup:** Don't run weekly distill for 3+ weeks (daily distills exist for each week).

**Steps:**
1. Ensure daily files exist spanning 3 ISO weeks
2. Set `last_weekly` in state to 3+ weeks ago (or delete state)
3. Run `ox distill --dry-run`

**Expected:**
- Output shows 3 weekly distillations planned
- Each week references only the daily files within that week's date range

4. Run `ox distill`

**Expected:**
- 3 weekly files: `memory/weekly/YYYY-W{XX}.md` for each week
- Each weekly file's footer lists only the daily files from its week
- `last_weekly` advanced to end of latest week

## Scenario 6: Monthly Catch-Up

**Setup:** Don't run monthly distill for 2+ months (weekly distills exist).

**Steps:**
1. Ensure weekly files exist spanning 2 months
2. Set `last_monthly` to 2+ months ago
3. Run `ox distill`

**Expected:**
- 2 monthly files: `memory/monthly/YYYY-MM.md` for each month
- Each monthly file references only weeklies overlapping that month

## Scenario 7: Discussion Facts Date Assignment

**Setup:** Discussion created on Day 1, distill run on Day 3.

**Steps:**
1. Create a discussion (via web UI) dated Day 1
2. Ensure the daemon syncs the discussion to team context
3. Run `ox distill` on Day 3

**Expected:**
- Discussion facts extracted into `memory/.discussion-facts/{dirname}.md`
- The daily summary that includes these facts is for Day 1 (the discussion's `created_at`), NOT Day 3
- Fact file footer contains `(created YYYY-MM-DD)` matching Day 1
