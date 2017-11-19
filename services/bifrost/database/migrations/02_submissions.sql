-- +migrate Up

ALTER TABLE transactions_queue
  ADD COLUMN locked_until TIMESTAMP;

ALTER TABLE transactions_queue
  ADD COLUMN locked_token VARCHAR(16);


CREATE TYPE SUBMISSION_TYPE AS ENUM ('submission_create_account', 'submission_send_tokens');

CREATE TABLE transaction_submissions (
  id                    BIGSERIAL,
  transactions_queue_id BIGINT          NOT NULL REFERENCES transactions_queue (id),
  type                  SUBMISSION_TYPE NOT NULL,
  created_at            TIMESTAMP       NOT NULL

);

-- +migrate Down

ALTER TABLE transactions_queue
  DROP COLUMN locked_until;

ALTER TABLE transactions_queue
  DROP COLUMN locked_token;


DROP TABLE transaction_submissions CASCADE;
DROP TYPE SUBMISSION_TYPE;
