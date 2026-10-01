-- Drop the station profile change time; nothing else depends on it.

ALTER TABLE endpoints DROP COLUMN profile_changed_at;
