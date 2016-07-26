-- Add a last_stop flag to scheduled_stop_time
ALTER TABLE scheduled_stop_time ADD COLUMN last_stop BOOLEAN;
