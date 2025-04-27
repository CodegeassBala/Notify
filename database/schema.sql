--
-- PostgreSQL database dump
--

-- Dumped from database version 17.4 (Debian 17.4-1.pgdg120+2)
-- Dumped by pg_dump version 17.4 (Debian 17.4-1.pgdg120+2)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: hstore; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "hstore" WITH SCHEMA "public";


--
-- Name: set_serial_id_from_clients(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION "public"."set_serial_id_from_clients"() RETURNS "trigger"
    LANGUAGE "plpgsql"
    AS $$
BEGIN
    -- Fetch the serial ID from the clients table
    SELECT id INTO NEW.serial_id FROM clients WHERE client_id = NEW.client_id;

    -- If client_id does not exist, raise an error
    IF NEW.serial_id IS NULL THEN
        RAISE EXCEPTION 'client_id % does not exist in clients table', NEW.client_id;
    END IF;

    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = "heap";

--
-- Name: clients; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE "public"."clients" (
    "id" integer NOT NULL,
    "client_id" "text" NOT NULL,
    "email" "text",
    "phone" "text",
    "connection" "text"
);


--
-- Name: clients_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE "public"."clients_id_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: clients_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE "public"."clients_id_seq" OWNED BY "public"."clients"."id";


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE "public"."schema_migrations" (
    "version" bigint NOT NULL,
    "dirty" boolean NOT NULL
);


--
-- Name: topics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE "public"."topics" (
    "topic_name" "text" NOT NULL
);


--
-- Name: topics_clients; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE "public"."topics_clients" (
    "client_id" "text" NOT NULL,
    "topic_name" "text" NOT NULL,
    "serial_id" integer NOT NULL
);


--
-- Name: variables; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE "public"."variables" (
    "h" "public"."hstore"
);


--
-- Name: clients id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY "public"."clients" ALTER COLUMN "id" SET DEFAULT "nextval"('"public"."clients_id_seq"'::"regclass");


--
-- Name: clients clients_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY "public"."clients"
    ADD CONSTRAINT "clients_id_key" UNIQUE ("id");


--
-- Name: clients clients_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY "public"."clients"
    ADD CONSTRAINT "clients_pkey" PRIMARY KEY ("client_id");


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY "public"."schema_migrations"
    ADD CONSTRAINT "schema_migrations_pkey" PRIMARY KEY ("version");


--
-- Name: topics topics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY "public"."topics"
    ADD CONSTRAINT "topics_pkey" PRIMARY KEY ("topic_name");


--
-- Name: topics_clients trigger_set_serial_id; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER "trigger_set_serial_id" BEFORE INSERT ON "public"."topics_clients" FOR EACH ROW EXECUTE FUNCTION "public"."set_serial_id_from_clients"();


--
-- Name: topics_clients topics_clients_client_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY "public"."topics_clients"
    ADD CONSTRAINT "topics_clients_client_id_fkey" FOREIGN KEY ("client_id") REFERENCES "public"."clients"("client_id") ON DELETE CASCADE;


--
-- Name: topics_clients topics_clients_topic_name_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY "public"."topics_clients"
    ADD CONSTRAINT "topics_clients_topic_name_fkey" FOREIGN KEY ("topic_name") REFERENCES "public"."topics"("topic_name") ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

