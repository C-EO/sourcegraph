-- This migration was generated by the command `sg telemetry add`
INSERT INTO event_logs_export_allowlist (event_name) VALUES (UNNEST('{AccessRequestsPageViewed,AccessRequestFailed,CreateAccessRequestSucceeded,AccessRequestRejected,AccessRequestApproved,SearchInputAdd,SearchInputGoto,SearchInputCommand}'::TEXT[])) ON CONFLICT DO NOTHING;
