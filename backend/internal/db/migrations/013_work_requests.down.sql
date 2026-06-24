-- +migrate Down
-- 013_work_requests.down.sql — откат WorkRequest таблицы
DROP TABLE IF EXISTS work_requests;
