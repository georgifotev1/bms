-- Debug query to identify why GetAvailableTimeslots returns empty results
-- Run this with your actual parameters to see what's happening at each step

WITH
-- Parameters: Replace these with your actual values
debug_params AS (
    SELECT
        1::INTEGER as brand_id,
        '2025-01-15'::DATE as date,
        'your-service-uuid'::UUID as service_id,
        1::BIGINT as user_id
),
-- Check if service exists and is visible
service_info AS (
    SELECT
        s.id,
        s.title,
        s.duration,
        COALESCE(s.buffer_time, 0) as buffer_time,
        s.is_visible,
        s.brand_id
    FROM services s, debug_params dp
    WHERE s.id = dp.service_id
),
-- Get working hours for the specific day
working_hours AS (
    SELECT
        bwh.brand_id,
        bwh.day_of_week,
        bwh.open_time,
        bwh.close_time,
        bwh.is_closed,
        EXTRACT(DOW FROM dp.date) as requested_dow
    FROM brand_working_hours bwh, debug_params dp
    WHERE bwh.brand_id = dp.brand_id
      AND bwh.day_of_week = EXTRACT(DOW FROM dp.date)
),
-- Get all events for the given date and selected user
daily_events AS (
    SELECT
        e.id,
        e.start_time,
        e.end_time,
        e.end_time + (INTERVAL '1 minute' * COALESCE(e.buffer_time, 0)) AS effective_end_time,
        e.user_id,
        e.brand_id
    FROM events e, debug_params dp
    WHERE e.brand_id = dp.brand_id
      AND DATE(e.start_time) = dp.date
      AND e.user_id = dp.user_id
),
-- Check if the selected staff member can perform this service
qualified_staff AS (
    SELECT
        u.id,
        u.name,
        u.verified,
        u.brand_id,
        us.service_id,
        s.title as service_title
    FROM users u
    JOIN user_services us ON u.id = us.user_id
    JOIN services s ON us.service_id = s.id,
    debug_params dp
    WHERE u.brand_id = dp.brand_id
      AND u.verified = true
      AND us.service_id = dp.service_id
      AND u.id = dp.user_id
),
-- Try to generate time slots
time_slots AS (
    SELECT
        generate_series(
            (dp.date + wh.open_time)::timestamp,
            (dp.date + wh.close_time - (INTERVAL '1 minute' * si.duration))::timestamp,
            INTERVAL '15 minutes'
        )::timestamp AS slot_start,
        si.duration,
        si.buffer_time
    FROM working_hours wh
    CROSS JOIN service_info si
    CROSS JOIN debug_params dp
    WHERE NOT wh.is_closed
      AND EXISTS (SELECT 1 FROM qualified_staff)
    LIMIT 5 -- Just show first 5 slots for debugging
)
-- Debug output - show what's happening at each step
SELECT 'Parameters' as step, 1 as sort_order, json_build_object(
    'brand_id', dp.brand_id,
    'date', dp.date,
    'service_id', dp.service_id,
    'user_id', dp.user_id,
    'day_of_week', EXTRACT(DOW FROM dp.date)
) as data
FROM debug_params dp

UNION ALL

SELECT 'Service Info' as step, 2 as sort_order,
CASE
    WHEN COUNT(*) > 0 THEN json_agg(si.*)
    ELSE '[]'::json
END as data
FROM service_info si

UNION ALL

SELECT 'Working Hours' as step, 3 as sort_order,
CASE
    WHEN COUNT(*) > 0 THEN json_agg(wh.*)
    ELSE '[]'::json
END as data
FROM working_hours wh

UNION ALL

SELECT 'Qualified Staff' as step, 4 as sort_order,
CASE
    WHEN COUNT(*) > 0 THEN json_agg(qs.*)
    ELSE '[]'::json
END as data
FROM qualified_staff qs

UNION ALL

SELECT 'Daily Events' as step, 5 as sort_order,
CASE
    WHEN COUNT(*) > 0 THEN json_agg(de.*)
    ELSE '[]'::json
END as data
FROM daily_events de

UNION ALL

SELECT 'Time Slots Generated' as step, 6 as sort_order,
CASE
    WHEN COUNT(*) > 0 THEN json_agg(ts.*)
    ELSE '[]'::json
END as data
FROM time_slots ts

ORDER BY sort_order;
