ALTER TABLE  topics_clients DROP CONSTRAINT IF EXISTS topics_clients_client_id_fkey;
ALTER TABLE  topics_clients DROP CONSTRAINT IF EXISTS topics_clients_topic_name_fkey;
DROP TABLE IF EXISTS topics_clients;