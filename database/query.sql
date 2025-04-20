-- name: InsertClient :exec
INSERT INTO "clients" (clientID, email, phone,connection)
VALUES ($1, $2, $3,$4);

-- name: InsertTopic :exec
INSERT INTO "topics" (topicName)
VALUES ($1);

-- name: InsertTopicClient :exec
INSERT INTO "topics_clients" (clientID, topicName)
VALUES ($1, $2);

-- name: DeleteTopicClient :exec
DELETE FROM "topics_clients"
WHERE clientID = $1 AND topicName = $2;
 
-- name: GetTopic :one
SELECT * FROM topics WHERE topicName = $1;