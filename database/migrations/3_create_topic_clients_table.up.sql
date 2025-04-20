CREATE TABLE IF NOT EXISTS topics_clients (
    clientID    TEXT NOT NULL,
    topicName   TEXT NOT NULL,
    serialId    INT NOT NULL,  
    FOREIGN KEY (clientID) REFERENCES clients(clientID) ON DELETE CASCADE,
    FOREIGN KEY (topicName) REFERENCES topics(topicName) ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION set_serial_id_from_clients()
RETURNS TRIGGER AS $$
BEGIN
    -- Fetch the serial ID from the clients table
    SELECT id INTO NEW.serialId FROM clients WHERE clientID = NEW.clientID;

    -- If clientID does not exist, raise an error
    IF NEW.serialId IS NULL THEN
        RAISE EXCEPTION 'clientID % does not exist in clients table', NEW.clientID;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_set_serial_id
BEFORE INSERT ON topics_clients
FOR EACH ROW
EXECUTE FUNCTION set_serial_id_from_clients();


