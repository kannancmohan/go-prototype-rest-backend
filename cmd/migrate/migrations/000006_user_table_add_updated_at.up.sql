ALTER TABLE
  users
ADD
  COLUMN updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW();