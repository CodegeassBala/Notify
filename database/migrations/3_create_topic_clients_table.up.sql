CREATE TABLE IF NOT EXISTS topics_clients (
    client_id    TEXT NOT NULL,
    topic_name   TEXT NOT NULL,
    serial_id    INT NOT NULL,  
    FOREIGN KEY (client_id) REFERENCES clients(client_id) ON DELETE CASCADE,
    FOREIGN KEY (topic_name) REFERENCES topics(topic_name) ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION set_serial_id_from_clients()
RETURNS TRIGGER AS $$
BEGIN
    -- Fetch the serial ID from the clients table
    SELECT id INTO NEW.serial_id FROM clients WHERE client_id = NEW.client_id;

    -- If client_id does not exist, raise an error
    IF NEW.serial_id IS NULL THEN
        RAISE EXCEPTION 'client_id % does not exist in clients table', NEW.client_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_serial_id
BEFORE INSERT ON topics_clients
FOR EACH ROW
EXECUTE FUNCTION set_serial_id_from_clients();


