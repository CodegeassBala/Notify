-- name: InsertClient :exec
INSERT INTO "clients" (client_id, email, phone,connection)
VALUES ($1, $2, $3,$4);

-- name: InsertTopic :exec
INSERT INTO "topics" (topic_name)
VALUES ($1);

-- name: InsertTopicClient :exec
INSERT INTO "topics_clients" (client_id, topic_name)
VALUES ($1, $2);

-- name: DeleteTopicClient :exec
DELETE FROM "topics_clients"
WHERE client_id = $1 AND topic_name = $2;
 
-- name: GetTopic :one
SELECT * FROM topics WHERE topic_name = $1;

-- name: GetAllTopics :many
SELECT * FROM topics;

-- name: GetConsumerQueuesCount :one 
SELECT h['client_consumers_count']::int AS client_consumers_count FROM variables;
 
-- name: GetClientsInRange :many
SELECT * FROM clients 
WHERE clients.client_id IN (
  SELECT client_id 
  FROM topics_clients 
  WHERE topic_name = @topic_name 
    AND id >= @start_id 
    AND id <= @end_id
);

-- name: SetConsumerQueuesCount :exec
UPDATE variables SET h = jsonb_set(h, '{client_consumers_count}', to_jsonb(@count::int));