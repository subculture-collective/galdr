--
-- PostgreSQL database dump
--

\restrict NKAhvuYSMdrGNf40aXMBAMdnUAhlcRc0KCV0BdoTXbTMhJat3iUUggWmawaMKAv

-- Dumped from database version 16.12
-- Dumped by pg_dump version 16.12

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: citext; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;


--
-- Name: EXTENSION citext; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION citext IS 'data type for case-insensitive character strings';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: uuid-ossp; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public;


--
-- Name: EXTENSION "uuid-ossp"; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION "uuid-ossp" IS 'generate universally unique identifiers (UUIDs)';


--
-- Name: trigger_set_updated_at(); Type: FUNCTION; Schema: public; Owner: pulsescore
--

CREATE FUNCTION public.trigger_set_updated_at() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.trigger_set_updated_at() OWNER TO pulsescore;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: alert_history; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.alert_history (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    alert_rule_id uuid NOT NULL,
    customer_id uuid,
    trigger_data jsonb,
    channel character varying(50) NOT NULL,
    status character varying(20) NOT NULL,
    sent_at timestamp with time zone,
    error_message text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    sendgrid_message_id character varying(255),
    delivered_at timestamp with time zone,
    opened_at timestamp with time zone,
    clicked_at timestamp with time zone,
    bounced_at timestamp with time zone,
    CONSTRAINT alert_history_status_check CHECK (((status)::text = ANY ((ARRAY['sent'::character varying, 'failed'::character varying, 'pending'::character varying])::text[])))
);


ALTER TABLE public.alert_history OWNER TO pulsescore;

--
-- Name: alert_rules; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.alert_rules (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    description text,
    trigger_type character varying(50) NOT NULL,
    conditions jsonb NOT NULL,
    channel character varying(50) DEFAULT 'email'::character varying NOT NULL,
    recipients jsonb,
    is_active boolean DEFAULT true NOT NULL,
    created_by uuid,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.alert_rules OWNER TO pulsescore;

--
-- Name: billing_webhook_events; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.billing_webhook_events (
    event_id character varying(255) NOT NULL,
    event_type character varying(120) NOT NULL,
    processed_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.billing_webhook_events OWNER TO pulsescore;

--
-- Name: customer_events; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.customer_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    event_type character varying(100) NOT NULL,
    source character varying(50) NOT NULL,
    external_event_id character varying(255),
    occurred_at timestamp with time zone NOT NULL,
    data jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.customer_events OWNER TO pulsescore;

--
-- Name: customers; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.customers (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    external_id character varying(255),
    source character varying(50) NOT NULL,
    email public.citext,
    name character varying(255),
    company_name character varying(255),
    mrr_cents integer DEFAULT 0 NOT NULL,
    currency character varying(3) DEFAULT 'USD'::character varying NOT NULL,
    first_seen_at timestamp with time zone,
    last_seen_at timestamp with time zone,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


ALTER TABLE public.customers OWNER TO pulsescore;

--
-- Name: health_score_history; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.health_score_history (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    overall_score integer NOT NULL,
    risk_level character varying(20) NOT NULL,
    factors jsonb NOT NULL,
    calculated_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT health_score_history_overall_score_check CHECK (((overall_score >= 0) AND (overall_score <= 100))),
    CONSTRAINT health_score_history_risk_level_check CHECK (((risk_level)::text = ANY ((ARRAY['green'::character varying, 'yellow'::character varying, 'red'::character varying])::text[])))
);


ALTER TABLE public.health_score_history OWNER TO pulsescore;

--
-- Name: health_scores; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.health_scores (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    overall_score integer NOT NULL,
    risk_level character varying(20) NOT NULL,
    factors jsonb NOT NULL,
    calculated_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT health_scores_overall_score_check CHECK (((overall_score >= 0) AND (overall_score <= 100))),
    CONSTRAINT health_scores_risk_level_check CHECK (((risk_level)::text = ANY ((ARRAY['green'::character varying, 'yellow'::character varying, 'red'::character varying])::text[])))
);


ALTER TABLE public.health_scores OWNER TO pulsescore;

--
-- Name: hubspot_companies; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.hubspot_companies (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    org_id uuid NOT NULL,
    hubspot_company_id character varying(255) NOT NULL,
    name character varying(500),
    domain character varying(500),
    industry character varying(255),
    number_of_employees integer,
    annual_revenue_cents bigint DEFAULT 0,
    metadata jsonb DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.hubspot_companies OWNER TO pulsescore;

--
-- Name: hubspot_contacts; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.hubspot_contacts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    org_id uuid NOT NULL,
    customer_id uuid,
    hubspot_contact_id character varying(255) NOT NULL,
    email public.citext,
    first_name character varying(255),
    last_name character varying(255),
    hubspot_company_id character varying(255),
    lifecycle_stage character varying(100),
    lead_status character varying(100),
    metadata jsonb DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.hubspot_contacts OWNER TO pulsescore;

--
-- Name: hubspot_deals; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.hubspot_deals (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    org_id uuid NOT NULL,
    customer_id uuid,
    hubspot_deal_id character varying(255) NOT NULL,
    hubspot_contact_id character varying(255),
    deal_name character varying(500),
    stage character varying(255),
    amount_cents bigint DEFAULT 0,
    currency character varying(3) DEFAULT 'USD'::character varying,
    close_date timestamp with time zone,
    pipeline character varying(255),
    metadata jsonb DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.hubspot_deals OWNER TO pulsescore;

--
-- Name: integration_connections; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.integration_connections (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    provider character varying(50) NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    access_token_encrypted bytea,
    refresh_token_encrypted bytea,
    token_expires_at timestamp with time zone,
    external_account_id character varying(255),
    scopes text[],
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    last_sync_at timestamp with time zone,
    last_sync_error text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT integration_connections_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'active'::character varying, 'error'::character varying, 'disconnected'::character varying])::text[])))
);


ALTER TABLE public.integration_connections OWNER TO pulsescore;

--
-- Name: intercom_contacts; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.intercom_contacts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    org_id uuid NOT NULL,
    customer_id uuid,
    intercom_contact_id character varying(255) NOT NULL,
    email public.citext,
    name character varying(500),
    role character varying(100),
    intercom_company_id character varying(255),
    metadata jsonb DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.intercom_contacts OWNER TO pulsescore;

--
-- Name: intercom_conversations; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.intercom_conversations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    org_id uuid NOT NULL,
    customer_id uuid,
    intercom_conversation_id character varying(255) NOT NULL,
    intercom_contact_id character varying(255),
    state character varying(100),
    rating integer,
    rating_remark text,
    open boolean DEFAULT true,
    read boolean DEFAULT false,
    priority character varying(50),
    subject character varying(1000),
    created_at_remote timestamp with time zone,
    updated_at_remote timestamp with time zone,
    closed_at timestamp with time zone,
    first_response_at timestamp with time zone,
    metadata jsonb DEFAULT '{}'::jsonb,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.intercom_conversations OWNER TO pulsescore;

--
-- Name: invitations; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.invitations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    email public.citext NOT NULL,
    role character varying(20) DEFAULT 'member'::character varying NOT NULL,
    token character varying(255) NOT NULL,
    status character varying(20) DEFAULT 'pending'::character varying NOT NULL,
    invited_by uuid NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT invitations_role_check CHECK (((role)::text = ANY ((ARRAY['owner'::character varying, 'admin'::character varying, 'member'::character varying])::text[]))),
    CONSTRAINT invitations_status_check CHECK (((status)::text = ANY ((ARRAY['pending'::character varying, 'accepted'::character varying, 'expired'::character varying])::text[])))
);


ALTER TABLE public.invitations OWNER TO pulsescore;

--
-- Name: notification_preferences; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.notification_preferences (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    org_id uuid NOT NULL,
    email_enabled boolean DEFAULT true NOT NULL,
    in_app_enabled boolean DEFAULT true NOT NULL,
    digest_enabled boolean DEFAULT false NOT NULL,
    digest_frequency character varying(20) DEFAULT 'weekly'::character varying NOT NULL,
    muted_rule_ids uuid[] DEFAULT '{}'::uuid[],
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT notification_preferences_digest_frequency_check CHECK (((digest_frequency)::text = ANY ((ARRAY['daily'::character varying, 'weekly'::character varying])::text[])))
);


ALTER TABLE public.notification_preferences OWNER TO pulsescore;

--
-- Name: notifications; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.notifications (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    org_id uuid NOT NULL,
    type character varying(50) NOT NULL,
    title character varying(255) NOT NULL,
    message text DEFAULT ''::text NOT NULL,
    data jsonb DEFAULT '{}'::jsonb,
    read_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.notifications OWNER TO pulsescore;

--
-- Name: onboarding_events; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.onboarding_events (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    step_id character varying(50) NOT NULL,
    event_type character varying(50) NOT NULL,
    occurred_at timestamp with time zone DEFAULT now() NOT NULL,
    duration_ms bigint,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT onboarding_events_event_type_check CHECK (((event_type)::text = ANY ((ARRAY['step_started'::character varying, 'step_completed'::character varying, 'step_skipped'::character varying, 'onboarding_completed'::character varying, 'onboarding_abandoned'::character varying])::text[])))
);


ALTER TABLE public.onboarding_events OWNER TO pulsescore;

--
-- Name: onboarding_status; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.onboarding_status (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    current_step character varying(50) DEFAULT 'welcome'::character varying NOT NULL,
    completed_steps text[] DEFAULT '{}'::text[] NOT NULL,
    skipped_steps text[] DEFAULT '{}'::text[] NOT NULL,
    step_payloads jsonb DEFAULT '{}'::jsonb NOT NULL,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.onboarding_status OWNER TO pulsescore;

--
-- Name: org_subscriptions; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.org_subscriptions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    stripe_subscription_id character varying(255),
    stripe_customer_id character varying(255),
    plan_tier character varying(50) DEFAULT 'free'::character varying NOT NULL,
    billing_cycle character varying(20) DEFAULT 'monthly'::character varying NOT NULL,
    status character varying(50) DEFAULT 'inactive'::character varying NOT NULL,
    current_period_start timestamp with time zone,
    current_period_end timestamp with time zone,
    cancel_at_period_end boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT org_subscriptions_billing_cycle_check CHECK (((billing_cycle)::text = ANY ((ARRAY['monthly'::character varying, 'annual'::character varying])::text[])))
);


ALTER TABLE public.org_subscriptions OWNER TO pulsescore;

--
-- Name: organizations; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.organizations (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    name character varying(255) NOT NULL,
    slug public.citext NOT NULL,
    plan character varying(50) DEFAULT 'free'::character varying NOT NULL,
    stripe_customer_id character varying(255),
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


ALTER TABLE public.organizations OWNER TO pulsescore;

--
-- Name: password_resets; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.password_resets (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.password_resets OWNER TO pulsescore;

--
-- Name: refresh_tokens; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.refresh_tokens (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    token_hash character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.refresh_tokens OWNER TO pulsescore;

--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO pulsescore;

--
-- Name: scoring_configs; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.scoring_configs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    org_id uuid NOT NULL,
    weights jsonb DEFAULT '{"mrr_trend": 0.2, "engagement": 0.15, "failed_payments": 0.2, "payment_recency": 0.3, "support_tickets": 0.15}'::jsonb NOT NULL,
    thresholds jsonb DEFAULT '{"green": 70, "yellow": 40}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.scoring_configs OWNER TO pulsescore;

--
-- Name: stripe_payments; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.stripe_payments (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    stripe_payment_id character varying(255) NOT NULL,
    amount_cents integer NOT NULL,
    currency character varying(3) DEFAULT 'USD'::character varying NOT NULL,
    status character varying(50) NOT NULL,
    failure_code character varying(100),
    failure_message text,
    paid_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.stripe_payments OWNER TO pulsescore;

--
-- Name: stripe_subscriptions; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.stripe_subscriptions (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    org_id uuid NOT NULL,
    customer_id uuid NOT NULL,
    stripe_subscription_id character varying(255) NOT NULL,
    status character varying(50) NOT NULL,
    plan_name character varying(255),
    amount_cents integer DEFAULT 0 NOT NULL,
    currency character varying(3) DEFAULT 'USD'::character varying NOT NULL,
    "interval" character varying(20),
    current_period_start timestamp with time zone,
    current_period_end timestamp with time zone,
    canceled_at timestamp with time zone,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


ALTER TABLE public.stripe_subscriptions OWNER TO pulsescore;

--
-- Name: user_organizations; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.user_organizations (
    user_id uuid NOT NULL,
    org_id uuid NOT NULL,
    role character varying(20) DEFAULT 'member'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_organizations_role_check CHECK (((role)::text = ANY ((ARRAY['owner'::character varying, 'admin'::character varying, 'member'::character varying])::text[])))
);


ALTER TABLE public.user_organizations OWNER TO pulsescore;

--
-- Name: users; Type: TABLE; Schema: public; Owner: pulsescore
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    email public.citext NOT NULL,
    password_hash character varying(255) NOT NULL,
    first_name character varying(100),
    last_name character varying(100),
    avatar_url text,
    email_verified boolean DEFAULT false NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);


ALTER TABLE public.users OWNER TO pulsescore;

--
-- Data for Name: alert_history; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.alert_history (id, org_id, alert_rule_id, customer_id, trigger_data, channel, status, sent_at, error_message, created_at, sendgrid_message_id, delivered_at, opened_at, clicked_at, bounced_at) FROM stdin;
\.


--
-- Data for Name: alert_rules; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.alert_rules (id, org_id, name, description, trigger_type, conditions, channel, recipients, is_active, created_by, created_at, updated_at) FROM stdin;
f0000000-0000-4000-a000-000000000001	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	Score Drop Alert	Alert when a customer health score drops below 50	score_drop	{"direction": "below", "threshold": 50}	email	["owner@acme.com", "admin@acme.com"]	t	b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01	2026-02-26 03:44:52.00195+00	2026-02-26 03:44:52.00195+00
f0000000-0000-4000-a000-000000000002	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	Payment Failed Alert	Alert when a customer payment fails	payment_failed	{"consecutive_failures": 1}	email	["owner@acme.com"]	t	b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01	2026-02-26 03:44:52.00195+00	2026-02-26 03:44:52.00195+00
\.


--
-- Data for Name: billing_webhook_events; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.billing_webhook_events (event_id, event_type, processed_at) FROM stdin;
\.


--
-- Data for Name: customer_events; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.customer_events (id, org_id, customer_id, event_type, source, external_event_id, occurred_at, data, created_at) FROM stdin;
577ae577-c68f-4228-a81a-4c33460fa314	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	subscription.created	stripe	evt_1_1	2026-02-16 02:54:51.919299+00	{"demo": true, "amount_cents": 150}	2026-02-26 03:44:51.919299+00
0401748c-8ae4-45ff-b532-fd789a517ef6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	subscription.renewed	stripe	evt_1_2	2026-02-07 02:41:51.919299+00	{"demo": true, "amount_cents": 200}	2026-02-26 03:44:51.919299+00
4ab5ccfd-e188-462b-9edc-fcae0105b2d9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	subscription.cancelled	stripe	evt_1_3	2026-01-29 02:28:51.919299+00	{"demo": true, "amount_cents": 250}	2026-02-26 03:44:51.919299+00
a49dfcba-a68f-4a3f-9d7a-964561b63714	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	ticket.opened	stripe	evt_1_4	2026-01-20 02:15:51.919299+00	{"demo": true, "amount_cents": 300}	2026-02-26 03:44:51.919299+00
c58fb6ed-cc62-4394-9279-6027e85548f5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	ticket.resolved	stripe	evt_1_5	2026-01-11 02:02:51.919299+00	{"demo": true, "amount_cents": 350}	2026-02-26 03:44:51.919299+00
acd946f7-ace3-4913-a134-04215c022bee	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	login	stripe	evt_1_6	2026-01-02 01:49:51.919299+00	{"demo": true, "amount_cents": 400}	2026-02-26 03:44:51.919299+00
51f6a042-21cc-4cbd-9c36-edbeeb1c5c76	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	feature.used	stripe	evt_1_7	2025-12-24 01:36:51.919299+00	{"demo": true, "amount_cents": 450}	2026-02-26 03:44:51.919299+00
2f4ac3d5-9016-40d7-af4f-a1579ca0203d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	payment.success	stripe	evt_1_8	2025-12-15 01:23:51.919299+00	{"demo": true, "amount_cents": 500}	2026-02-26 03:44:51.919299+00
ca5ed26b-1a95-4b39-abed-9878ee78d2d5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	payment.failed	stripe	evt_1_9	2025-12-06 01:10:51.919299+00	{"demo": true, "amount_cents": 550}	2026-02-26 03:44:51.919299+00
50850c99-3d99-4f1f-8e2a-bbc0c2024685	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	subscription.created	stripe	evt_1_10	2025-11-27 00:57:51.919299+00	{"demo": true, "amount_cents": 600}	2026-02-26 03:44:51.919299+00
ec912c7e-8061-41cf-9612-018550be6048	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	subscription.renewed	stripe	evt_2_1	2026-02-15 02:17:51.919299+00	{"demo": true, "amount_cents": 250}	2026-02-26 03:44:51.919299+00
7dfa949f-cd8f-4dc5-8d2d-3fa95711d893	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	subscription.cancelled	stripe	evt_2_2	2026-02-06 02:04:51.919299+00	{"demo": true, "amount_cents": 300}	2026-02-26 03:44:51.919299+00
c5e7a482-743c-4782-ab2f-7015cbb0bcf2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	ticket.opened	stripe	evt_2_3	2026-01-28 01:51:51.919299+00	{"demo": true, "amount_cents": 350}	2026-02-26 03:44:51.919299+00
ad2e8e70-780b-4872-b455-4d933729adc8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	ticket.resolved	stripe	evt_2_4	2026-01-19 01:38:51.919299+00	{"demo": true, "amount_cents": 400}	2026-02-26 03:44:51.919299+00
9886e9f9-7aae-49a7-b0b1-194f75483e92	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	login	stripe	evt_2_5	2026-01-10 01:25:51.919299+00	{"demo": true, "amount_cents": 450}	2026-02-26 03:44:51.919299+00
ddee5ec5-9ba5-4137-a625-ccac9ebb26c7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	feature.used	stripe	evt_2_6	2026-01-01 01:12:51.919299+00	{"demo": true, "amount_cents": 500}	2026-02-26 03:44:51.919299+00
06cc0d37-c9f0-4ab7-95a3-1e5c28e8cc79	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	payment.success	stripe	evt_2_7	2025-12-23 00:59:51.919299+00	{"demo": true, "amount_cents": 550}	2026-02-26 03:44:51.919299+00
c57a9d69-b126-4f5d-bd9d-4d555cc11d3e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	payment.failed	stripe	evt_2_8	2025-12-14 00:46:51.919299+00	{"demo": true, "amount_cents": 600}	2026-02-26 03:44:51.919299+00
2237fb8a-4184-4ba2-bdf4-0cd98aaebfce	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	subscription.created	stripe	evt_2_9	2025-12-05 00:33:51.919299+00	{"demo": true, "amount_cents": 650}	2026-02-26 03:44:51.919299+00
c0f5e614-2151-4677-a75f-d5ee41a4c374	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	subscription.renewed	stripe	evt_2_10	2025-11-26 00:20:51.919299+00	{"demo": true, "amount_cents": 700}	2026-02-26 03:44:51.919299+00
39d0f70d-ae82-4d61-893c-545599bef6ea	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	subscription.cancelled	stripe	evt_3_1	2026-02-14 01:40:51.919299+00	{"demo": true, "amount_cents": 350}	2026-02-26 03:44:51.919299+00
e5730c63-40d8-4425-b8b3-5ada20c78ef6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	ticket.opened	stripe	evt_3_2	2026-02-05 01:27:51.919299+00	{"demo": true, "amount_cents": 400}	2026-02-26 03:44:51.919299+00
c383cc1c-af38-4331-9b1f-ea5f8225989d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	ticket.resolved	stripe	evt_3_3	2026-01-27 01:14:51.919299+00	{"demo": true, "amount_cents": 450}	2026-02-26 03:44:51.919299+00
ecfbec1b-0943-4976-9f82-7f5b575a2cc1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	login	stripe	evt_3_4	2026-01-18 01:01:51.919299+00	{"demo": true, "amount_cents": 500}	2026-02-26 03:44:51.919299+00
49127306-672d-4984-9957-6bd75d2445a4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	feature.used	stripe	evt_3_5	2026-01-09 00:48:51.919299+00	{"demo": true, "amount_cents": 550}	2026-02-26 03:44:51.919299+00
9b68a3f3-07da-4243-a082-7af0a972d76d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	payment.success	stripe	evt_3_6	2025-12-31 00:35:51.919299+00	{"demo": true, "amount_cents": 600}	2026-02-26 03:44:51.919299+00
817c3bb1-c5e1-45ba-9f23-055650639947	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	payment.failed	stripe	evt_3_7	2025-12-22 00:22:51.919299+00	{"demo": true, "amount_cents": 650}	2026-02-26 03:44:51.919299+00
31e49dcb-0f53-4a38-bfed-12436be2d2af	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	subscription.created	stripe	evt_3_8	2025-12-13 00:09:51.919299+00	{"demo": true, "amount_cents": 700}	2026-02-26 03:44:51.919299+00
9159d544-0647-4767-aa7a-5e0990a97569	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	subscription.renewed	stripe	evt_3_9	2025-12-03 23:56:51.919299+00	{"demo": true, "amount_cents": 750}	2026-02-26 03:44:51.919299+00
3190137b-7d7c-4d69-940a-1a51811713da	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	subscription.cancelled	stripe	evt_3_10	2025-11-24 23:43:51.919299+00	{"demo": true, "amount_cents": 800}	2026-02-26 03:44:51.919299+00
37d43830-5782-4e50-ae3e-088343e558ca	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	ticket.opened	stripe	evt_4_1	2026-02-13 01:03:51.919299+00	{"demo": true, "amount_cents": 450}	2026-02-26 03:44:51.919299+00
fc6acdb8-3b68-4598-82e9-39e52c45466f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	ticket.resolved	stripe	evt_4_2	2026-02-04 00:50:51.919299+00	{"demo": true, "amount_cents": 500}	2026-02-26 03:44:51.919299+00
1114bb1d-00aa-4a7f-94e9-1604fa957359	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	login	stripe	evt_4_3	2026-01-26 00:37:51.919299+00	{"demo": true, "amount_cents": 550}	2026-02-26 03:44:51.919299+00
efa5a6e1-2090-4d06-bb07-f7b56ed34c35	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	feature.used	stripe	evt_4_4	2026-01-17 00:24:51.919299+00	{"demo": true, "amount_cents": 600}	2026-02-26 03:44:51.919299+00
1d9c2021-128a-4f65-8fe3-4d3cb7bb9f28	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	payment.success	stripe	evt_4_5	2026-01-08 00:11:51.919299+00	{"demo": true, "amount_cents": 650}	2026-02-26 03:44:51.919299+00
d527335f-fad2-41ee-a243-21cbedaf2240	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	payment.failed	stripe	evt_4_6	2025-12-29 23:58:51.919299+00	{"demo": true, "amount_cents": 700}	2026-02-26 03:44:51.919299+00
93b3cf57-9e3f-41da-b98c-3c739c5f6667	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	subscription.created	stripe	evt_4_7	2025-12-20 23:45:51.919299+00	{"demo": true, "amount_cents": 750}	2026-02-26 03:44:51.919299+00
440dfb14-d735-4880-818e-27d1eda2f090	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	subscription.renewed	stripe	evt_4_8	2025-12-11 23:32:51.919299+00	{"demo": true, "amount_cents": 800}	2026-02-26 03:44:51.919299+00
b5913c01-c1a5-46f0-8fd8-f6c8da1f8593	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	subscription.cancelled	stripe	evt_4_9	2025-12-02 23:19:51.919299+00	{"demo": true, "amount_cents": 850}	2026-02-26 03:44:51.919299+00
48bee954-d177-4bc7-8b84-a9e9fc75bf54	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	ticket.opened	stripe	evt_4_10	2025-11-23 23:06:51.919299+00	{"demo": true, "amount_cents": 900}	2026-02-26 03:44:51.919299+00
60604f6d-43c6-4479-9681-251267d0b16a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	ticket.resolved	stripe	evt_5_1	2026-02-17 00:26:51.919299+00	{"demo": true, "amount_cents": 550}	2026-02-26 03:44:51.919299+00
7d9c31b5-e8b2-47f7-a41a-8f7bb3e2b881	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	login	stripe	evt_5_2	2026-02-08 00:13:51.919299+00	{"demo": true, "amount_cents": 600}	2026-02-26 03:44:51.919299+00
ae3c258a-f542-4643-b9ad-bdc495823f98	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	feature.used	stripe	evt_5_3	2026-01-30 00:00:51.919299+00	{"demo": true, "amount_cents": 650}	2026-02-26 03:44:51.919299+00
950e375c-f2a1-4fdd-b7ff-183cebd69a57	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	payment.success	stripe	evt_5_4	2026-01-20 23:47:51.919299+00	{"demo": true, "amount_cents": 700}	2026-02-26 03:44:51.919299+00
83973e6c-6eda-413b-ad0c-adee96ed36ef	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	payment.failed	stripe	evt_5_5	2026-01-11 23:34:51.919299+00	{"demo": true, "amount_cents": 750}	2026-02-26 03:44:51.919299+00
69dc3243-471e-4670-97d8-e8795c8d451d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	subscription.created	stripe	evt_5_6	2026-01-02 23:21:51.919299+00	{"demo": true, "amount_cents": 800}	2026-02-26 03:44:51.919299+00
5bc04b24-3709-4035-9a36-332dcef87abe	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	subscription.renewed	stripe	evt_5_7	2025-12-24 23:08:51.919299+00	{"demo": true, "amount_cents": 850}	2026-02-26 03:44:51.919299+00
ca86484c-00c4-4716-a096-f95af888cca7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	subscription.cancelled	stripe	evt_5_8	2025-12-15 22:55:51.919299+00	{"demo": true, "amount_cents": 900}	2026-02-26 03:44:51.919299+00
8a8d2c93-82a5-4066-bcc5-8a1399874113	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	ticket.opened	stripe	evt_5_9	2025-12-06 22:42:51.919299+00	{"demo": true, "amount_cents": 950}	2026-02-26 03:44:51.919299+00
b083ffd4-8692-4216-ad2d-8c70f51a5cdf	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	ticket.resolved	stripe	evt_5_10	2025-11-27 22:29:51.919299+00	{"demo": true, "amount_cents": 1000}	2026-02-26 03:44:51.919299+00
52bb08b7-0ea6-44ce-8d14-d629703edbaa	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	login	stripe	evt_6_1	2026-02-15 23:49:51.919299+00	{"demo": true, "amount_cents": 650}	2026-02-26 03:44:51.919299+00
47baf6e8-7d98-4e62-afa0-6c57a70345b7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	feature.used	stripe	evt_6_2	2026-02-06 23:36:51.919299+00	{"demo": true, "amount_cents": 700}	2026-02-26 03:44:51.919299+00
fd4953fd-b573-4dd1-be54-dbb828e7db7b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	payment.success	stripe	evt_6_3	2026-01-28 23:23:51.919299+00	{"demo": true, "amount_cents": 750}	2026-02-26 03:44:51.919299+00
8d261165-5f49-44e4-87fd-efcdbbf063c1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	payment.failed	stripe	evt_6_4	2026-01-19 23:10:51.919299+00	{"demo": true, "amount_cents": 800}	2026-02-26 03:44:51.919299+00
7fe4c064-2188-454a-9f40-db9a3ab8b586	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	subscription.created	stripe	evt_6_5	2026-01-10 22:57:51.919299+00	{"demo": true, "amount_cents": 850}	2026-02-26 03:44:51.919299+00
b4371350-04ea-4645-a275-8f5add0da870	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	subscription.renewed	stripe	evt_6_6	2026-01-01 22:44:51.919299+00	{"demo": true, "amount_cents": 900}	2026-02-26 03:44:51.919299+00
599f0faf-49d8-4e6e-bde7-22d1dbd6f76d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	subscription.cancelled	stripe	evt_6_7	2025-12-23 22:31:51.919299+00	{"demo": true, "amount_cents": 950}	2026-02-26 03:44:51.919299+00
7de79e57-8012-419a-9bf3-62496aa9920b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	ticket.opened	stripe	evt_6_8	2025-12-14 22:18:51.919299+00	{"demo": true, "amount_cents": 1000}	2026-02-26 03:44:51.919299+00
b410b494-94dd-41c1-a05d-1b98cbc76725	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	ticket.resolved	stripe	evt_6_9	2025-12-05 22:05:51.919299+00	{"demo": true, "amount_cents": 1050}	2026-02-26 03:44:51.919299+00
d9eb220a-a1cf-4d18-8acf-44dc80a15694	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	login	stripe	evt_6_10	2025-11-26 21:52:51.919299+00	{"demo": true, "amount_cents": 1100}	2026-02-26 03:44:51.919299+00
ebf76f14-f7b4-4225-add9-dc3cd183260f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	feature.used	stripe	evt_7_1	2026-02-14 23:12:51.919299+00	{"demo": true, "amount_cents": 750}	2026-02-26 03:44:51.919299+00
1bd90567-86db-4812-aa9d-867326b6ab14	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	payment.success	stripe	evt_7_2	2026-02-05 22:59:51.919299+00	{"demo": true, "amount_cents": 800}	2026-02-26 03:44:51.919299+00
512e2cda-4302-415e-81bd-02ae878a3d62	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	payment.failed	stripe	evt_7_3	2026-01-27 22:46:51.919299+00	{"demo": true, "amount_cents": 850}	2026-02-26 03:44:51.919299+00
57612708-59fd-46ab-bd03-ecad227856ee	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	subscription.created	stripe	evt_7_4	2026-01-18 22:33:51.919299+00	{"demo": true, "amount_cents": 900}	2026-02-26 03:44:51.919299+00
32979d4d-1b4e-4079-b205-023c4205b05d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	subscription.renewed	stripe	evt_7_5	2026-01-09 22:20:51.919299+00	{"demo": true, "amount_cents": 950}	2026-02-26 03:44:51.919299+00
93ef881e-93b4-4c88-8979-6d2b2ff1856b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	subscription.cancelled	stripe	evt_7_6	2025-12-31 22:07:51.919299+00	{"demo": true, "amount_cents": 1000}	2026-02-26 03:44:51.919299+00
f11ff7ba-6d1c-47a9-880d-5c66cda49d96	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	ticket.opened	stripe	evt_7_7	2025-12-22 21:54:51.919299+00	{"demo": true, "amount_cents": 1050}	2026-02-26 03:44:51.919299+00
388f0af3-710d-4b71-822b-a36833a0b467	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	ticket.resolved	stripe	evt_7_8	2025-12-13 21:41:51.919299+00	{"demo": true, "amount_cents": 1100}	2026-02-26 03:44:51.919299+00
7eb93482-175c-47ea-b38c-b28bcdaaf0d3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	login	stripe	evt_7_9	2025-12-04 21:28:51.919299+00	{"demo": true, "amount_cents": 1150}	2026-02-26 03:44:51.919299+00
713f8698-0934-4f16-b95c-05755f217454	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	feature.used	stripe	evt_7_10	2025-11-25 21:15:51.919299+00	{"demo": true, "amount_cents": 1200}	2026-02-26 03:44:51.919299+00
03528a44-4037-4215-ba00-04fa42f27bbd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	payment.success	stripe	evt_8_1	2026-02-13 22:35:51.919299+00	{"demo": true, "amount_cents": 850}	2026-02-26 03:44:51.919299+00
f88e701f-7cfe-4467-a8cd-db97bbe8735c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	payment.failed	stripe	evt_8_2	2026-02-04 22:22:51.919299+00	{"demo": true, "amount_cents": 900}	2026-02-26 03:44:51.919299+00
51724114-b37d-4de0-ab58-96fea588f160	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	subscription.created	stripe	evt_8_3	2026-01-26 22:09:51.919299+00	{"demo": true, "amount_cents": 950}	2026-02-26 03:44:51.919299+00
704358b3-886e-4fed-8439-2717f2d9204a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	subscription.renewed	stripe	evt_8_4	2026-01-17 21:56:51.919299+00	{"demo": true, "amount_cents": 1000}	2026-02-26 03:44:51.919299+00
fe628107-8866-4431-8466-f253d2972f46	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	subscription.cancelled	stripe	evt_8_5	2026-01-08 21:43:51.919299+00	{"demo": true, "amount_cents": 1050}	2026-02-26 03:44:51.919299+00
8d502b9e-aec0-4574-806f-da2545aa612d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	ticket.opened	stripe	evt_8_6	2025-12-30 21:30:51.919299+00	{"demo": true, "amount_cents": 1100}	2026-02-26 03:44:51.919299+00
e89e67a3-b425-4fd1-9e44-1be5091de79f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	ticket.resolved	stripe	evt_8_7	2025-12-21 21:17:51.919299+00	{"demo": true, "amount_cents": 1150}	2026-02-26 03:44:51.919299+00
ab7c113e-60e9-4c54-8e70-e74a706b9c76	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	login	stripe	evt_8_8	2025-12-12 21:04:51.919299+00	{"demo": true, "amount_cents": 1200}	2026-02-26 03:44:51.919299+00
a6ff529a-fec1-433a-b934-5c0686787289	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	feature.used	stripe	evt_8_9	2025-12-03 20:51:51.919299+00	{"demo": true, "amount_cents": 1250}	2026-02-26 03:44:51.919299+00
817538d2-7fe9-4553-b563-3afd66c96c6c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	payment.success	stripe	evt_8_10	2025-11-24 20:38:51.919299+00	{"demo": true, "amount_cents": 1300}	2026-02-26 03:44:51.919299+00
0e10a38b-a828-4abb-be4e-4efacb3901f7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	payment.failed	stripe	evt_9_1	2026-02-12 21:58:51.919299+00	{"demo": true, "amount_cents": 950}	2026-02-26 03:44:51.919299+00
1d70d1da-a320-440b-8dd0-6ffa0016c57d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	subscription.created	stripe	evt_9_2	2026-02-03 21:45:51.919299+00	{"demo": true, "amount_cents": 1000}	2026-02-26 03:44:51.919299+00
42a059f3-fe48-427d-8acb-96155b8b5fe2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	subscription.renewed	stripe	evt_9_3	2026-01-25 21:32:51.919299+00	{"demo": true, "amount_cents": 1050}	2026-02-26 03:44:51.919299+00
d70db3d1-1e92-46fc-b94a-51f6920963b0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	subscription.cancelled	stripe	evt_9_4	2026-01-16 21:19:51.919299+00	{"demo": true, "amount_cents": 1100}	2026-02-26 03:44:51.919299+00
e2e167c0-19c1-4b05-a722-4aa3cc5a18a0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	ticket.opened	stripe	evt_9_5	2026-01-07 21:06:51.919299+00	{"demo": true, "amount_cents": 1150}	2026-02-26 03:44:51.919299+00
d9742afd-9035-4476-8e96-b9d13330fd3c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	ticket.resolved	stripe	evt_9_6	2025-12-29 20:53:51.919299+00	{"demo": true, "amount_cents": 1200}	2026-02-26 03:44:51.919299+00
9f6566c7-2333-496b-b12c-af0dfb9581b8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	login	stripe	evt_9_7	2025-12-20 20:40:51.919299+00	{"demo": true, "amount_cents": 1250}	2026-02-26 03:44:51.919299+00
961c870d-6f91-4e66-85db-0807b1a70f54	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	feature.used	stripe	evt_9_8	2025-12-11 20:27:51.919299+00	{"demo": true, "amount_cents": 1300}	2026-02-26 03:44:51.919299+00
34b2e269-3f05-4953-9ffa-3837d99f9f99	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	payment.success	stripe	evt_9_9	2025-12-02 20:14:51.919299+00	{"demo": true, "amount_cents": 1350}	2026-02-26 03:44:51.919299+00
f0c1a562-0d1e-4225-918e-82c38bcffaf9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	payment.failed	stripe	evt_9_10	2025-11-23 20:01:51.919299+00	{"demo": true, "amount_cents": 1400}	2026-02-26 03:44:51.919299+00
c0e805d5-ab84-4ecc-b4ee-cdc2640383e7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	subscription.created	stripe	evt_10_1	2026-02-16 21:21:51.919299+00	{"demo": true, "amount_cents": 1050}	2026-02-26 03:44:51.919299+00
c352df9a-a4fd-4f25-823f-a56fd24ebdbc	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	subscription.renewed	stripe	evt_10_2	2026-02-07 21:08:51.919299+00	{"demo": true, "amount_cents": 1100}	2026-02-26 03:44:51.919299+00
afc37f87-1832-41e9-9604-86be102fca9f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	subscription.cancelled	stripe	evt_10_3	2026-01-29 20:55:51.919299+00	{"demo": true, "amount_cents": 1150}	2026-02-26 03:44:51.919299+00
42eaae9b-7e3f-4203-bae3-db15f82380e5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	ticket.opened	stripe	evt_10_4	2026-01-20 20:42:51.919299+00	{"demo": true, "amount_cents": 1200}	2026-02-26 03:44:51.919299+00
1091eb67-66dc-4835-97e1-218b6d7ac689	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	ticket.resolved	stripe	evt_10_5	2026-01-11 20:29:51.919299+00	{"demo": true, "amount_cents": 1250}	2026-02-26 03:44:51.919299+00
519891c5-492d-4a0f-ab70-7359f969f87c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	login	stripe	evt_10_6	2026-01-02 20:16:51.919299+00	{"demo": true, "amount_cents": 1300}	2026-02-26 03:44:51.919299+00
ec8e36cd-8d78-43a8-939b-c7314dd1c378	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	feature.used	stripe	evt_10_7	2025-12-24 20:03:51.919299+00	{"demo": true, "amount_cents": 1350}	2026-02-26 03:44:51.919299+00
cab2b065-f053-47d9-9cc6-5104c3ebed7f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	payment.success	stripe	evt_10_8	2025-12-15 19:50:51.919299+00	{"demo": true, "amount_cents": 1400}	2026-02-26 03:44:51.919299+00
c5459e08-b9d5-445c-b690-1af3399ad2d6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	payment.failed	stripe	evt_10_9	2025-12-06 19:37:51.919299+00	{"demo": true, "amount_cents": 1450}	2026-02-26 03:44:51.919299+00
5ae1cd10-fb0b-4477-8d08-ae0808e2bdc6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	subscription.created	stripe	evt_10_10	2025-11-27 19:24:51.919299+00	{"demo": true, "amount_cents": 1500}	2026-02-26 03:44:51.919299+00
60abd0a1-4083-4424-a1f6-a59bbb94f6ca	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	subscription.renewed	stripe	evt_11_1	2026-02-15 20:44:51.919299+00	{"demo": true, "amount_cents": 1150}	2026-02-26 03:44:51.919299+00
eb27b1ef-b0ab-4cc3-b255-1918be67857e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	subscription.cancelled	stripe	evt_11_2	2026-02-06 20:31:51.919299+00	{"demo": true, "amount_cents": 1200}	2026-02-26 03:44:51.919299+00
553a480e-b3e8-488f-b616-09bb4625a58e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	ticket.opened	stripe	evt_11_3	2026-01-28 20:18:51.919299+00	{"demo": true, "amount_cents": 1250}	2026-02-26 03:44:51.919299+00
b2b7f309-ca5a-47c1-b122-0dfdd01f3d67	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	ticket.resolved	stripe	evt_11_4	2026-01-19 20:05:51.919299+00	{"demo": true, "amount_cents": 1300}	2026-02-26 03:44:51.919299+00
85cd4231-adc4-4f3b-a837-49806a95d9b3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	login	stripe	evt_11_5	2026-01-10 19:52:51.919299+00	{"demo": true, "amount_cents": 1350}	2026-02-26 03:44:51.919299+00
afa6c482-5c11-4192-bf98-e13e67a2f89e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	feature.used	stripe	evt_11_6	2026-01-01 19:39:51.919299+00	{"demo": true, "amount_cents": 1400}	2026-02-26 03:44:51.919299+00
83fdf448-912b-4350-b159-5a464c5d04a1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	payment.success	stripe	evt_11_7	2025-12-23 19:26:51.919299+00	{"demo": true, "amount_cents": 1450}	2026-02-26 03:44:51.919299+00
b16f729c-c8a9-472c-9c3e-ceaeffe732fd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	payment.failed	stripe	evt_11_8	2025-12-14 19:13:51.919299+00	{"demo": true, "amount_cents": 1500}	2026-02-26 03:44:51.919299+00
07978371-49a1-45a5-ac30-35ea5d7b9c25	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	subscription.created	stripe	evt_11_9	2025-12-05 19:00:51.919299+00	{"demo": true, "amount_cents": 1550}	2026-02-26 03:44:51.919299+00
b1bd06e6-4fe9-4706-8908-555c69a3366f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	subscription.renewed	stripe	evt_11_10	2025-11-26 18:47:51.919299+00	{"demo": true, "amount_cents": 1600}	2026-02-26 03:44:51.919299+00
b8899a97-9d32-48c3-8d7f-fd7d07884003	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	subscription.cancelled	stripe	evt_12_1	2026-02-14 20:07:51.919299+00	{"demo": true, "amount_cents": 1250}	2026-02-26 03:44:51.919299+00
94d5728f-8b7a-4da1-947b-23fc46143dff	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	ticket.opened	stripe	evt_12_2	2026-02-05 19:54:51.919299+00	{"demo": true, "amount_cents": 1300}	2026-02-26 03:44:51.919299+00
9b598daf-91fd-4714-8500-61b961b426d1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	ticket.resolved	stripe	evt_12_3	2026-01-27 19:41:51.919299+00	{"demo": true, "amount_cents": 1350}	2026-02-26 03:44:51.919299+00
9976483b-613c-4de6-8e0d-562b2e855cce	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	login	stripe	evt_12_4	2026-01-18 19:28:51.919299+00	{"demo": true, "amount_cents": 1400}	2026-02-26 03:44:51.919299+00
6cae8690-2c32-4e44-a3b0-32ab1c8ca8f0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	feature.used	stripe	evt_12_5	2026-01-09 19:15:51.919299+00	{"demo": true, "amount_cents": 1450}	2026-02-26 03:44:51.919299+00
5863e598-a281-4b12-894a-ea0693fbea5e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	payment.success	stripe	evt_12_6	2025-12-31 19:02:51.919299+00	{"demo": true, "amount_cents": 1500}	2026-02-26 03:44:51.919299+00
60328d1f-ac0f-4eee-9ae5-ce8ee717bb68	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	payment.failed	stripe	evt_12_7	2025-12-22 18:49:51.919299+00	{"demo": true, "amount_cents": 1550}	2026-02-26 03:44:51.919299+00
1b62c91b-ddf3-4381-8ce4-83f03c59f9d8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	subscription.created	stripe	evt_12_8	2025-12-13 18:36:51.919299+00	{"demo": true, "amount_cents": 1600}	2026-02-26 03:44:51.919299+00
3a6aa79c-d291-4219-a368-fb0e5abea57f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	subscription.renewed	stripe	evt_12_9	2025-12-04 18:23:51.919299+00	{"demo": true, "amount_cents": 1650}	2026-02-26 03:44:51.919299+00
a6c2db69-596f-4aa4-9b7f-48c637998bc6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	subscription.cancelled	stripe	evt_12_10	2025-11-25 18:10:51.919299+00	{"demo": true, "amount_cents": 1700}	2026-02-26 03:44:51.919299+00
96b678f2-636d-4f82-9c1c-da62fc5596e1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	ticket.opened	stripe	evt_13_1	2026-02-13 19:30:51.919299+00	{"demo": true, "amount_cents": 1350}	2026-02-26 03:44:51.919299+00
1a05551f-a385-4ea7-8fb6-0da50b852b8e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	ticket.resolved	stripe	evt_13_2	2026-02-04 19:17:51.919299+00	{"demo": true, "amount_cents": 1400}	2026-02-26 03:44:51.919299+00
7f80b383-32e9-48aa-97db-82e515f91e42	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	login	stripe	evt_13_3	2026-01-26 19:04:51.919299+00	{"demo": true, "amount_cents": 1450}	2026-02-26 03:44:51.919299+00
14bd4ceb-7dfe-4ca3-950e-a43e872c940b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	feature.used	stripe	evt_13_4	2026-01-17 18:51:51.919299+00	{"demo": true, "amount_cents": 1500}	2026-02-26 03:44:51.919299+00
c1d485e7-a192-46c2-8f2f-f8b4acaf22b5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	payment.success	stripe	evt_13_5	2026-01-08 18:38:51.919299+00	{"demo": true, "amount_cents": 1550}	2026-02-26 03:44:51.919299+00
61adc9d6-12e6-4def-9ee4-d758eb26f9e1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	payment.failed	stripe	evt_13_6	2025-12-30 18:25:51.919299+00	{"demo": true, "amount_cents": 1600}	2026-02-26 03:44:51.919299+00
40086547-416a-4d6b-99c1-8bcf4face787	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	subscription.created	stripe	evt_13_7	2025-12-21 18:12:51.919299+00	{"demo": true, "amount_cents": 1650}	2026-02-26 03:44:51.919299+00
bc3577e0-7de3-458b-8b3d-00c107ccc226	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	subscription.renewed	stripe	evt_13_8	2025-12-12 17:59:51.919299+00	{"demo": true, "amount_cents": 1700}	2026-02-26 03:44:51.919299+00
4cd90e8b-f8ec-4ff1-a6fc-7529ae40768d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	subscription.cancelled	stripe	evt_13_9	2025-12-03 17:46:51.919299+00	{"demo": true, "amount_cents": 1750}	2026-02-26 03:44:51.919299+00
befc088f-61fb-42d4-92a7-37751059a9f3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	ticket.opened	stripe	evt_13_10	2025-11-24 17:33:51.919299+00	{"demo": true, "amount_cents": 1800}	2026-02-26 03:44:51.919299+00
72482afd-8884-4792-a387-07f8132d031c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	ticket.resolved	stripe	evt_14_1	2026-02-12 18:53:51.919299+00	{"demo": true, "amount_cents": 1450}	2026-02-26 03:44:51.919299+00
570e1de4-a07b-47ed-8411-f950000e6484	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	login	stripe	evt_14_2	2026-02-03 18:40:51.919299+00	{"demo": true, "amount_cents": 1500}	2026-02-26 03:44:51.919299+00
57e87d23-df78-4fcb-bf85-bb5a3847003c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	feature.used	stripe	evt_14_3	2026-01-25 18:27:51.919299+00	{"demo": true, "amount_cents": 1550}	2026-02-26 03:44:51.919299+00
a233eeb6-5f56-423e-81d2-f5c3afc26dde	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	payment.success	stripe	evt_14_4	2026-01-16 18:14:51.919299+00	{"demo": true, "amount_cents": 1600}	2026-02-26 03:44:51.919299+00
1dda453a-190e-4493-9ffc-a2a02697c3ba	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	payment.failed	stripe	evt_14_5	2026-01-07 18:01:51.919299+00	{"demo": true, "amount_cents": 1650}	2026-02-26 03:44:51.919299+00
b3872e82-fc5e-428f-84de-17bfe25f5a10	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	subscription.created	stripe	evt_14_6	2025-12-29 17:48:51.919299+00	{"demo": true, "amount_cents": 1700}	2026-02-26 03:44:51.919299+00
0804e16e-8c97-43f9-96e4-ee045a59df49	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	subscription.renewed	stripe	evt_14_7	2025-12-20 17:35:51.919299+00	{"demo": true, "amount_cents": 1750}	2026-02-26 03:44:51.919299+00
c5e9a25e-7f8a-46be-91cf-3a7cce46a06c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	subscription.cancelled	stripe	evt_14_8	2025-12-11 17:22:51.919299+00	{"demo": true, "amount_cents": 1800}	2026-02-26 03:44:51.919299+00
2f9bfcd8-4f82-4f72-bfed-49406e43ff81	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	ticket.opened	stripe	evt_14_9	2025-12-02 17:09:51.919299+00	{"demo": true, "amount_cents": 1850}	2026-02-26 03:44:51.919299+00
c312a1d0-7574-46f9-aa51-2de3a73a4085	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	ticket.resolved	stripe	evt_14_10	2025-11-23 16:56:51.919299+00	{"demo": true, "amount_cents": 1900}	2026-02-26 03:44:51.919299+00
3a5c6228-c23f-472b-84d2-f495dbe7f2c2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	login	stripe	evt_15_1	2026-02-16 18:16:51.919299+00	{"demo": true, "amount_cents": 1550}	2026-02-26 03:44:51.919299+00
3beb8f1e-a43f-4087-8f3c-e3a730625fb6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	feature.used	stripe	evt_15_2	2026-02-07 18:03:51.919299+00	{"demo": true, "amount_cents": 1600}	2026-02-26 03:44:51.919299+00
f83e621c-ee41-41ba-b11c-6a1240aacd83	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	payment.success	stripe	evt_15_3	2026-01-29 17:50:51.919299+00	{"demo": true, "amount_cents": 1650}	2026-02-26 03:44:51.919299+00
8a1171b3-5d6b-48c4-85bf-284931b8d645	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	payment.failed	stripe	evt_15_4	2026-01-20 17:37:51.919299+00	{"demo": true, "amount_cents": 1700}	2026-02-26 03:44:51.919299+00
878d31fa-8de6-4346-bf3c-24ad0b21c775	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	subscription.created	stripe	evt_15_5	2026-01-11 17:24:51.919299+00	{"demo": true, "amount_cents": 1750}	2026-02-26 03:44:51.919299+00
89c69090-6a29-45df-afb6-49fa2495a131	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	subscription.renewed	stripe	evt_15_6	2026-01-02 17:11:51.919299+00	{"demo": true, "amount_cents": 1800}	2026-02-26 03:44:51.919299+00
23a497e9-9cc6-466f-abf7-14ff05044ad9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	subscription.cancelled	stripe	evt_15_7	2025-12-24 16:58:51.919299+00	{"demo": true, "amount_cents": 1850}	2026-02-26 03:44:51.919299+00
b62e14d4-0157-475b-abc4-4d1f48479b34	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	ticket.opened	stripe	evt_15_8	2025-12-15 16:45:51.919299+00	{"demo": true, "amount_cents": 1900}	2026-02-26 03:44:51.919299+00
5fbbf08d-077c-4889-827e-c79a397d2522	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	ticket.resolved	stripe	evt_15_9	2025-12-06 16:32:51.919299+00	{"demo": true, "amount_cents": 1950}	2026-02-26 03:44:51.919299+00
2254a31b-36b2-4baa-801e-0a6e9586d441	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	login	stripe	evt_15_10	2025-11-27 16:19:51.919299+00	{"demo": true, "amount_cents": 2000}	2026-02-26 03:44:51.919299+00
fddb3231-bc40-4d48-93d5-a245329ed581	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	feature.used	stripe	evt_16_1	2026-02-15 17:39:51.919299+00	{"demo": true, "amount_cents": 1650}	2026-02-26 03:44:51.919299+00
78a4f87d-b59a-4c35-bf46-c1cf0c1677dd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	payment.success	stripe	evt_16_2	2026-02-06 17:26:51.919299+00	{"demo": true, "amount_cents": 1700}	2026-02-26 03:44:51.919299+00
3402efdf-b281-4a34-9e60-2d5f326ae70b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	payment.failed	stripe	evt_16_3	2026-01-28 17:13:51.919299+00	{"demo": true, "amount_cents": 1750}	2026-02-26 03:44:51.919299+00
55c01470-e2b6-4934-a54d-079a71bb386f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	subscription.created	stripe	evt_16_4	2026-01-19 17:00:51.919299+00	{"demo": true, "amount_cents": 1800}	2026-02-26 03:44:51.919299+00
fe8a37c7-0863-457a-b04b-d05c6795218f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	subscription.renewed	stripe	evt_16_5	2026-01-10 16:47:51.919299+00	{"demo": true, "amount_cents": 1850}	2026-02-26 03:44:51.919299+00
a3623d37-f6a8-4fc5-b779-c13e19087772	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	subscription.cancelled	stripe	evt_16_6	2026-01-01 16:34:51.919299+00	{"demo": true, "amount_cents": 1900}	2026-02-26 03:44:51.919299+00
dbf92f68-8f66-43a3-9164-93c9de48759f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	ticket.opened	stripe	evt_16_7	2025-12-23 16:21:51.919299+00	{"demo": true, "amount_cents": 1950}	2026-02-26 03:44:51.919299+00
978e4fb3-3e7d-42d4-a2d9-2ce8a41267c8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	ticket.resolved	stripe	evt_16_8	2025-12-14 16:08:51.919299+00	{"demo": true, "amount_cents": 2000}	2026-02-26 03:44:51.919299+00
5e7fb821-70ad-42bd-ad1a-afe5f1dff41b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	login	stripe	evt_16_9	2025-12-05 15:55:51.919299+00	{"demo": true, "amount_cents": 2050}	2026-02-26 03:44:51.919299+00
28433cfc-876f-4fab-be28-074565649b65	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	feature.used	stripe	evt_16_10	2025-11-26 15:42:51.919299+00	{"demo": true, "amount_cents": 2100}	2026-02-26 03:44:51.919299+00
37ddf688-a3d0-4653-a282-54b1b7a99c12	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	payment.success	stripe	evt_17_1	2026-02-14 17:02:51.919299+00	{"demo": true, "amount_cents": 1750}	2026-02-26 03:44:51.919299+00
58c8013d-3bcf-4f5f-9b28-d9552b7524c2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	payment.failed	stripe	evt_17_2	2026-02-05 16:49:51.919299+00	{"demo": true, "amount_cents": 1800}	2026-02-26 03:44:51.919299+00
b93b72de-bee6-487a-aa11-1a3d86fea139	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	subscription.created	stripe	evt_17_3	2026-01-27 16:36:51.919299+00	{"demo": true, "amount_cents": 1850}	2026-02-26 03:44:51.919299+00
83420a4d-76a9-4e10-ae26-a2e78dedaef5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	subscription.renewed	stripe	evt_17_4	2026-01-18 16:23:51.919299+00	{"demo": true, "amount_cents": 1900}	2026-02-26 03:44:51.919299+00
0a18acf2-9f31-4656-b610-8f6592eba6da	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	subscription.cancelled	stripe	evt_17_5	2026-01-09 16:10:51.919299+00	{"demo": true, "amount_cents": 1950}	2026-02-26 03:44:51.919299+00
0a8997a2-7b22-4102-949a-470c25cf0cec	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	ticket.opened	stripe	evt_17_6	2025-12-31 15:57:51.919299+00	{"demo": true, "amount_cents": 2000}	2026-02-26 03:44:51.919299+00
3fdb88fb-5734-4e6e-bbe0-19890811b0c7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	ticket.resolved	stripe	evt_17_7	2025-12-22 15:44:51.919299+00	{"demo": true, "amount_cents": 2050}	2026-02-26 03:44:51.919299+00
bd0a4416-014d-49ee-a88f-c7ac10f9d6d3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	login	stripe	evt_17_8	2025-12-13 15:31:51.919299+00	{"demo": true, "amount_cents": 2100}	2026-02-26 03:44:51.919299+00
9bcf2c55-8801-4052-b087-122a0241ae8f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	feature.used	stripe	evt_17_9	2025-12-04 15:18:51.919299+00	{"demo": true, "amount_cents": 2150}	2026-02-26 03:44:51.919299+00
b9a8c1c4-7e31-4a57-800c-d331f6172e71	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	payment.success	stripe	evt_17_10	2025-11-25 15:05:51.919299+00	{"demo": true, "amount_cents": 2200}	2026-02-26 03:44:51.919299+00
37903a5a-1544-4498-b5d1-6f9803d0e688	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	payment.failed	stripe	evt_18_1	2026-02-13 16:25:51.919299+00	{"demo": true, "amount_cents": 1850}	2026-02-26 03:44:51.919299+00
544d57ae-b82b-49d9-ab3a-fd258a898518	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	subscription.created	stripe	evt_18_2	2026-02-04 16:12:51.919299+00	{"demo": true, "amount_cents": 1900}	2026-02-26 03:44:51.919299+00
1341b6ce-febf-4a49-a5cf-0aad0831698f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	subscription.renewed	stripe	evt_18_3	2026-01-26 15:59:51.919299+00	{"demo": true, "amount_cents": 1950}	2026-02-26 03:44:51.919299+00
14f5835f-bbab-4234-96f9-1bd2957d617f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	subscription.cancelled	stripe	evt_18_4	2026-01-17 15:46:51.919299+00	{"demo": true, "amount_cents": 2000}	2026-02-26 03:44:51.919299+00
3a0a1c6b-e404-4d3c-b39c-bd4a3a3b78e3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	ticket.opened	stripe	evt_18_5	2026-01-08 15:33:51.919299+00	{"demo": true, "amount_cents": 2050}	2026-02-26 03:44:51.919299+00
0b50c8c3-be22-456c-96ed-17d2bb258d1f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	ticket.resolved	stripe	evt_18_6	2025-12-30 15:20:51.919299+00	{"demo": true, "amount_cents": 2100}	2026-02-26 03:44:51.919299+00
80d04da7-7b35-45aa-ae5b-8c566ff4f850	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	login	stripe	evt_18_7	2025-12-21 15:07:51.919299+00	{"demo": true, "amount_cents": 2150}	2026-02-26 03:44:51.919299+00
fb323de7-a5d1-4883-aaf2-a48aa73a966c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	feature.used	stripe	evt_18_8	2025-12-12 14:54:51.919299+00	{"demo": true, "amount_cents": 2200}	2026-02-26 03:44:51.919299+00
e6ceb68b-dcdf-491b-8cec-0534d8f35e26	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	payment.success	stripe	evt_18_9	2025-12-03 14:41:51.919299+00	{"demo": true, "amount_cents": 2250}	2026-02-26 03:44:51.919299+00
85b4d02d-dfdc-4ed3-8e9a-81463f3fc08d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	payment.failed	stripe	evt_18_10	2025-11-24 14:28:51.919299+00	{"demo": true, "amount_cents": 2300}	2026-02-26 03:44:51.919299+00
d5095fad-3dd1-4ceb-8429-621b00b29186	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	subscription.created	stripe	evt_19_1	2026-02-12 15:48:51.919299+00	{"demo": true, "amount_cents": 1950}	2026-02-26 03:44:51.919299+00
94bc1214-9ea3-4961-8a48-1be6d3227835	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	subscription.renewed	stripe	evt_19_2	2026-02-03 15:35:51.919299+00	{"demo": true, "amount_cents": 2000}	2026-02-26 03:44:51.919299+00
84156d5a-226c-4069-af11-8feaddb4bffe	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	subscription.cancelled	stripe	evt_19_3	2026-01-25 15:22:51.919299+00	{"demo": true, "amount_cents": 2050}	2026-02-26 03:44:51.919299+00
a362a363-90bf-48f7-8919-45831d3865ee	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	ticket.opened	stripe	evt_19_4	2026-01-16 15:09:51.919299+00	{"demo": true, "amount_cents": 2100}	2026-02-26 03:44:51.919299+00
85a33ad0-bad4-48fb-a56b-1e34ec5ad294	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	ticket.resolved	stripe	evt_19_5	2026-01-07 14:56:51.919299+00	{"demo": true, "amount_cents": 2150}	2026-02-26 03:44:51.919299+00
a9155191-7367-44ab-9ef4-c2a7d775c81b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	login	stripe	evt_19_6	2025-12-29 14:43:51.919299+00	{"demo": true, "amount_cents": 2200}	2026-02-26 03:44:51.919299+00
bd9c73a4-46fe-4912-a570-45925ba671a7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	feature.used	stripe	evt_19_7	2025-12-20 14:30:51.919299+00	{"demo": true, "amount_cents": 2250}	2026-02-26 03:44:51.919299+00
6eda8d21-3260-4875-82f4-b7b783db43ae	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	payment.success	stripe	evt_19_8	2025-12-11 14:17:51.919299+00	{"demo": true, "amount_cents": 2300}	2026-02-26 03:44:51.919299+00
e47d6c88-5dab-40be-ae1d-2456b3dcef2f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	payment.failed	stripe	evt_19_9	2025-12-02 14:04:51.919299+00	{"demo": true, "amount_cents": 2350}	2026-02-26 03:44:51.919299+00
68d33da2-5b7f-4913-8e02-904a48e4af50	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	subscription.created	stripe	evt_19_10	2025-11-23 13:51:51.919299+00	{"demo": true, "amount_cents": 2400}	2026-02-26 03:44:51.919299+00
a6ccba39-25b3-46e0-aea0-46b743b2842d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	subscription.renewed	stripe	evt_20_1	2026-02-16 15:11:51.919299+00	{"demo": true, "amount_cents": 2050}	2026-02-26 03:44:51.919299+00
f2e9f523-5dd3-4dfe-86a1-83e641a9208d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	subscription.cancelled	stripe	evt_20_2	2026-02-07 14:58:51.919299+00	{"demo": true, "amount_cents": 2100}	2026-02-26 03:44:51.919299+00
406eb252-0ece-424e-b9b8-900c3f0f329b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	ticket.opened	stripe	evt_20_3	2026-01-29 14:45:51.919299+00	{"demo": true, "amount_cents": 2150}	2026-02-26 03:44:51.919299+00
77e1b40c-c05f-460f-a8e4-e340ebea572d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	ticket.resolved	stripe	evt_20_4	2026-01-20 14:32:51.919299+00	{"demo": true, "amount_cents": 2200}	2026-02-26 03:44:51.919299+00
706f06fc-fb50-46c7-aa7c-31398473f52d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	login	stripe	evt_20_5	2026-01-11 14:19:51.919299+00	{"demo": true, "amount_cents": 2250}	2026-02-26 03:44:51.919299+00
d29046af-e934-4bac-bda6-012dd4028d24	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	feature.used	stripe	evt_20_6	2026-01-02 14:06:51.919299+00	{"demo": true, "amount_cents": 2300}	2026-02-26 03:44:51.919299+00
7692bfa2-d4ec-4523-852d-78e1996eafcb	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	payment.success	stripe	evt_20_7	2025-12-24 13:53:51.919299+00	{"demo": true, "amount_cents": 2350}	2026-02-26 03:44:51.919299+00
3fae1551-954c-432b-b703-99030964e307	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	payment.failed	stripe	evt_20_8	2025-12-15 13:40:51.919299+00	{"demo": true, "amount_cents": 2400}	2026-02-26 03:44:51.919299+00
e2fe0773-bdf3-484d-b0b5-24f46a006143	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	subscription.created	stripe	evt_20_9	2025-12-06 13:27:51.919299+00	{"demo": true, "amount_cents": 2450}	2026-02-26 03:44:51.919299+00
b69cbac6-d25a-43cb-b961-f1e734b78919	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	subscription.renewed	stripe	evt_20_10	2025-11-27 13:14:51.919299+00	{"demo": true, "amount_cents": 2500}	2026-02-26 03:44:51.919299+00
39aaa706-cee7-463d-8d66-84617d8824a3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	subscription.cancelled	stripe	evt_21_1	2026-02-15 14:34:51.919299+00	{"demo": true, "amount_cents": 2150}	2026-02-26 03:44:51.919299+00
c298a68a-fa19-4629-aa8e-fcfd4ff90fb5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	ticket.opened	stripe	evt_21_2	2026-02-06 14:21:51.919299+00	{"demo": true, "amount_cents": 2200}	2026-02-26 03:44:51.919299+00
979d56b4-92ad-40e8-b1f5-46e286420b8c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	ticket.resolved	stripe	evt_21_3	2026-01-28 14:08:51.919299+00	{"demo": true, "amount_cents": 2250}	2026-02-26 03:44:51.919299+00
d32605e7-da29-49e9-9447-4552121274ea	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	login	stripe	evt_21_4	2026-01-19 13:55:51.919299+00	{"demo": true, "amount_cents": 2300}	2026-02-26 03:44:51.919299+00
aeea5da1-b30d-4f6c-a39f-1e1e8116f477	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	feature.used	stripe	evt_21_5	2026-01-10 13:42:51.919299+00	{"demo": true, "amount_cents": 2350}	2026-02-26 03:44:51.919299+00
ff21e375-9179-4cd4-a71e-ef079ed9f753	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	payment.success	stripe	evt_21_6	2026-01-01 13:29:51.919299+00	{"demo": true, "amount_cents": 2400}	2026-02-26 03:44:51.919299+00
56ff1421-3bf4-4353-82dc-08303c32ed38	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	payment.failed	stripe	evt_21_7	2025-12-23 13:16:51.919299+00	{"demo": true, "amount_cents": 2450}	2026-02-26 03:44:51.919299+00
0519c3a0-e252-4a02-8ca7-988b1767316d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	subscription.created	stripe	evt_21_8	2025-12-14 13:03:51.919299+00	{"demo": true, "amount_cents": 2500}	2026-02-26 03:44:51.919299+00
a04de641-9c7a-4f0b-9036-22fe0e781d3e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	subscription.renewed	stripe	evt_21_9	2025-12-05 12:50:51.919299+00	{"demo": true, "amount_cents": 2550}	2026-02-26 03:44:51.919299+00
fda5c2e2-7d1f-43e9-9573-0bfbb3c50183	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	subscription.cancelled	stripe	evt_21_10	2025-11-26 12:37:51.919299+00	{"demo": true, "amount_cents": 2600}	2026-02-26 03:44:51.919299+00
bed062a3-fa84-41cd-8ffb-cd11a141426c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	ticket.opened	stripe	evt_22_1	2026-02-14 13:57:51.919299+00	{"demo": true, "amount_cents": 2250}	2026-02-26 03:44:51.919299+00
fb24cb18-1a91-443f-92b4-8da668393358	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	ticket.resolved	stripe	evt_22_2	2026-02-05 13:44:51.919299+00	{"demo": true, "amount_cents": 2300}	2026-02-26 03:44:51.919299+00
7a990240-cf0f-4837-bc75-544b885ce871	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	login	stripe	evt_22_3	2026-01-27 13:31:51.919299+00	{"demo": true, "amount_cents": 2350}	2026-02-26 03:44:51.919299+00
519f17cb-d8f6-4f68-ab51-142f542017e0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	feature.used	stripe	evt_22_4	2026-01-18 13:18:51.919299+00	{"demo": true, "amount_cents": 2400}	2026-02-26 03:44:51.919299+00
c89fe2c4-5926-4755-8261-d3414ef26f0c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	payment.success	stripe	evt_22_5	2026-01-09 13:05:51.919299+00	{"demo": true, "amount_cents": 2450}	2026-02-26 03:44:51.919299+00
98d0bbb3-feb0-4819-82ce-f719e5e10240	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	payment.failed	stripe	evt_22_6	2025-12-31 12:52:51.919299+00	{"demo": true, "amount_cents": 2500}	2026-02-26 03:44:51.919299+00
e61ca0cc-87b9-4d83-9b0b-0f50d20bb474	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	subscription.created	stripe	evt_22_7	2025-12-22 12:39:51.919299+00	{"demo": true, "amount_cents": 2550}	2026-02-26 03:44:51.919299+00
e788d626-30f7-4884-817a-62cc57bf30c4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	subscription.renewed	stripe	evt_22_8	2025-12-13 12:26:51.919299+00	{"demo": true, "amount_cents": 2600}	2026-02-26 03:44:51.919299+00
d018bbe1-11f4-4843-a0ee-de84ef5112ec	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	subscription.cancelled	stripe	evt_22_9	2025-12-04 12:13:51.919299+00	{"demo": true, "amount_cents": 2650}	2026-02-26 03:44:51.919299+00
2278be7d-04b0-4dce-92c1-10d28f4c1989	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	ticket.opened	stripe	evt_22_10	2025-11-25 12:00:51.919299+00	{"demo": true, "amount_cents": 2700}	2026-02-26 03:44:51.919299+00
7cd0d0ce-c6f1-4e1d-8f66-ba0f70cd2ac4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	ticket.resolved	stripe	evt_23_1	2026-02-13 13:20:51.919299+00	{"demo": true, "amount_cents": 2350}	2026-02-26 03:44:51.919299+00
cba04a02-0cf6-486c-b9c6-354a456132cf	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	login	stripe	evt_23_2	2026-02-04 13:07:51.919299+00	{"demo": true, "amount_cents": 2400}	2026-02-26 03:44:51.919299+00
e09868c0-f20b-468a-8a7e-a05e980cf06b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	feature.used	stripe	evt_23_3	2026-01-26 12:54:51.919299+00	{"demo": true, "amount_cents": 2450}	2026-02-26 03:44:51.919299+00
1d577fac-da9c-4236-9795-52832fede423	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	payment.success	stripe	evt_23_4	2026-01-17 12:41:51.919299+00	{"demo": true, "amount_cents": 2500}	2026-02-26 03:44:51.919299+00
8573ca1f-3d33-40fd-b22b-22c030177d8b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	payment.failed	stripe	evt_23_5	2026-01-08 12:28:51.919299+00	{"demo": true, "amount_cents": 2550}	2026-02-26 03:44:51.919299+00
9d78b927-24dd-4ba3-bb02-aea35bb57a9c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	subscription.created	stripe	evt_23_6	2025-12-30 12:15:51.919299+00	{"demo": true, "amount_cents": 2600}	2026-02-26 03:44:51.919299+00
e0752552-a03a-4063-975f-7011201a3d6c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	subscription.renewed	stripe	evt_23_7	2025-12-21 12:02:51.919299+00	{"demo": true, "amount_cents": 2650}	2026-02-26 03:44:51.919299+00
b22618c6-bdff-4e01-bc3d-37b2be1b6856	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	subscription.cancelled	stripe	evt_23_8	2025-12-12 11:49:51.919299+00	{"demo": true, "amount_cents": 2700}	2026-02-26 03:44:51.919299+00
bc0ae718-6b12-45e4-b40f-64d9211b60fe	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	ticket.opened	stripe	evt_23_9	2025-12-03 11:36:51.919299+00	{"demo": true, "amount_cents": 2750}	2026-02-26 03:44:51.919299+00
69e2a044-e33d-4462-b6e1-0ca9ac6526b3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	ticket.resolved	stripe	evt_23_10	2025-11-24 11:23:51.919299+00	{"demo": true, "amount_cents": 2800}	2026-02-26 03:44:51.919299+00
fb198670-9793-4bb2-9e8c-6ebf681897dc	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	login	stripe	evt_24_1	2026-02-12 12:43:51.919299+00	{"demo": true, "amount_cents": 2450}	2026-02-26 03:44:51.919299+00
879ea938-d5a5-4165-8b18-04830d637da7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	feature.used	stripe	evt_24_2	2026-02-03 12:30:51.919299+00	{"demo": true, "amount_cents": 2500}	2026-02-26 03:44:51.919299+00
6a90106c-dab8-4b3d-a1e5-b8df98412bfa	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	payment.success	stripe	evt_24_3	2026-01-25 12:17:51.919299+00	{"demo": true, "amount_cents": 2550}	2026-02-26 03:44:51.919299+00
e125de39-5494-4feb-90f7-c9cecb77ac7e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	payment.failed	stripe	evt_24_4	2026-01-16 12:04:51.919299+00	{"demo": true, "amount_cents": 2600}	2026-02-26 03:44:51.919299+00
cd9e0b94-024c-4137-a62a-ef82c3001437	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	subscription.created	stripe	evt_24_5	2026-01-07 11:51:51.919299+00	{"demo": true, "amount_cents": 2650}	2026-02-26 03:44:51.919299+00
00513697-c123-42b1-a5a7-100866b56b9e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	subscription.renewed	stripe	evt_24_6	2025-12-29 11:38:51.919299+00	{"demo": true, "amount_cents": 2700}	2026-02-26 03:44:51.919299+00
91e87202-3d19-4398-8e9d-35f90d5b1a8b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	subscription.cancelled	stripe	evt_24_7	2025-12-20 11:25:51.919299+00	{"demo": true, "amount_cents": 2750}	2026-02-26 03:44:51.919299+00
c0ba3e8f-5570-48f5-967a-1170d0edf8ef	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	ticket.opened	stripe	evt_24_8	2025-12-11 11:12:51.919299+00	{"demo": true, "amount_cents": 2800}	2026-02-26 03:44:51.919299+00
0769580e-17b0-4061-855a-ab0af9ff814c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	ticket.resolved	stripe	evt_24_9	2025-12-02 10:59:51.919299+00	{"demo": true, "amount_cents": 2850}	2026-02-26 03:44:51.919299+00
1565d976-ee62-453a-a364-902c967ea07f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	login	stripe	evt_24_10	2025-11-23 10:46:51.919299+00	{"demo": true, "amount_cents": 2900}	2026-02-26 03:44:51.919299+00
55a051dd-2f8f-4569-b54e-ed2413ac99ef	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	feature.used	stripe	evt_25_1	2026-02-16 12:06:51.919299+00	{"demo": true, "amount_cents": 2550}	2026-02-26 03:44:51.919299+00
1d4fddf3-9241-4a8e-8c86-03b41c45f620	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	payment.success	stripe	evt_25_2	2026-02-07 11:53:51.919299+00	{"demo": true, "amount_cents": 2600}	2026-02-26 03:44:51.919299+00
6d683e83-96a6-4202-9592-fd9b536dde3a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	payment.failed	stripe	evt_25_3	2026-01-29 11:40:51.919299+00	{"demo": true, "amount_cents": 2650}	2026-02-26 03:44:51.919299+00
a43313c2-3bb2-4e90-a456-0f46c553965d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	subscription.created	stripe	evt_25_4	2026-01-20 11:27:51.919299+00	{"demo": true, "amount_cents": 2700}	2026-02-26 03:44:51.919299+00
fcfd6d61-4e93-496b-bcf4-9e1f446c03a0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	subscription.renewed	stripe	evt_25_5	2026-01-11 11:14:51.919299+00	{"demo": true, "amount_cents": 2750}	2026-02-26 03:44:51.919299+00
83dd6688-9624-432d-9189-c1efc6141bc8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	subscription.cancelled	stripe	evt_25_6	2026-01-02 11:01:51.919299+00	{"demo": true, "amount_cents": 2800}	2026-02-26 03:44:51.919299+00
386af809-e2b9-47ac-b325-7ae7c3a87a54	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	ticket.opened	stripe	evt_25_7	2025-12-24 10:48:51.919299+00	{"demo": true, "amount_cents": 2850}	2026-02-26 03:44:51.919299+00
3c4e4640-fa12-4192-9e28-5ff09b426689	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	ticket.resolved	stripe	evt_25_8	2025-12-15 10:35:51.919299+00	{"demo": true, "amount_cents": 2900}	2026-02-26 03:44:51.919299+00
7fb82615-9b3b-406b-a674-c15c9f320507	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	login	stripe	evt_25_9	2025-12-06 10:22:51.919299+00	{"demo": true, "amount_cents": 2950}	2026-02-26 03:44:51.919299+00
ae5e631a-eef6-4ae1-a0f1-01b21ad3a2e9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	feature.used	stripe	evt_25_10	2025-11-27 10:09:51.919299+00	{"demo": true, "amount_cents": 3000}	2026-02-26 03:44:51.919299+00
e6b6aef8-15c0-4532-bb2f-65821d98dfca	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	payment.success	stripe	evt_26_1	2026-02-15 11:29:51.919299+00	{"demo": true, "amount_cents": 2650}	2026-02-26 03:44:51.919299+00
67a16815-25bd-47a3-923e-26face431405	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	payment.failed	stripe	evt_26_2	2026-02-06 11:16:51.919299+00	{"demo": true, "amount_cents": 2700}	2026-02-26 03:44:51.919299+00
d0cde3e4-56ef-44d8-ae02-b53f0fa8d380	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	subscription.created	stripe	evt_26_3	2026-01-28 11:03:51.919299+00	{"demo": true, "amount_cents": 2750}	2026-02-26 03:44:51.919299+00
eb491ca6-5518-4f42-a506-a078d6575b35	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	subscription.renewed	stripe	evt_26_4	2026-01-19 10:50:51.919299+00	{"demo": true, "amount_cents": 2800}	2026-02-26 03:44:51.919299+00
119d3b65-190f-4885-b754-52c7961e8dd3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	subscription.cancelled	stripe	evt_26_5	2026-01-10 10:37:51.919299+00	{"demo": true, "amount_cents": 2850}	2026-02-26 03:44:51.919299+00
08809e37-7d37-4494-b786-09f84762b441	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	ticket.opened	stripe	evt_26_6	2026-01-01 10:24:51.919299+00	{"demo": true, "amount_cents": 2900}	2026-02-26 03:44:51.919299+00
d246c5eb-ffde-47bb-a209-6cac4c9af9a9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	ticket.resolved	stripe	evt_26_7	2025-12-23 10:11:51.919299+00	{"demo": true, "amount_cents": 2950}	2026-02-26 03:44:51.919299+00
bc4efff5-02cb-476e-b560-69396c6aa910	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	login	stripe	evt_26_8	2025-12-14 09:58:51.919299+00	{"demo": true, "amount_cents": 3000}	2026-02-26 03:44:51.919299+00
09526b95-d964-4cff-bc6b-5bf80e851fa7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	feature.used	stripe	evt_26_9	2025-12-05 09:45:51.919299+00	{"demo": true, "amount_cents": 3050}	2026-02-26 03:44:51.919299+00
e73302e8-c066-4f36-82c7-d769e6949971	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	payment.success	stripe	evt_26_10	2025-11-26 09:32:51.919299+00	{"demo": true, "amount_cents": 3100}	2026-02-26 03:44:51.919299+00
15050f78-5344-4ef6-bfdc-1006d855a9ad	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	payment.failed	stripe	evt_27_1	2026-02-14 10:52:51.919299+00	{"demo": true, "amount_cents": 2750}	2026-02-26 03:44:51.919299+00
09243916-a75d-4358-813f-5f3b95ec4dc0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	subscription.created	stripe	evt_27_2	2026-02-05 10:39:51.919299+00	{"demo": true, "amount_cents": 2800}	2026-02-26 03:44:51.919299+00
eeccc7be-f607-4c1c-9500-0bf280d55a1b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	subscription.renewed	stripe	evt_27_3	2026-01-27 10:26:51.919299+00	{"demo": true, "amount_cents": 2850}	2026-02-26 03:44:51.919299+00
e29b9c0b-525b-449e-af49-67122e6d3045	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	subscription.cancelled	stripe	evt_27_4	2026-01-18 10:13:51.919299+00	{"demo": true, "amount_cents": 2900}	2026-02-26 03:44:51.919299+00
b24e2f2b-2c30-4281-942c-8a4fe0d40170	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	ticket.opened	stripe	evt_27_5	2026-01-09 10:00:51.919299+00	{"demo": true, "amount_cents": 2950}	2026-02-26 03:44:51.919299+00
d121acf4-ec89-453f-8e3d-085e3dc2e95a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	ticket.resolved	stripe	evt_27_6	2025-12-31 09:47:51.919299+00	{"demo": true, "amount_cents": 3000}	2026-02-26 03:44:51.919299+00
c85262bd-25e4-4a03-8b01-6368e4a51967	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	login	stripe	evt_27_7	2025-12-22 09:34:51.919299+00	{"demo": true, "amount_cents": 3050}	2026-02-26 03:44:51.919299+00
61370d41-9278-4584-a8b3-4a626320607a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	feature.used	stripe	evt_27_8	2025-12-13 09:21:51.919299+00	{"demo": true, "amount_cents": 3100}	2026-02-26 03:44:51.919299+00
8bfe0c04-ff2e-4bf1-a35c-4e2d2e008cc3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	payment.success	stripe	evt_27_9	2025-12-04 09:08:51.919299+00	{"demo": true, "amount_cents": 3150}	2026-02-26 03:44:51.919299+00
25004c47-487e-4dc6-b0f0-b0f62e196143	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	payment.failed	stripe	evt_27_10	2025-11-25 08:55:51.919299+00	{"demo": true, "amount_cents": 3200}	2026-02-26 03:44:51.919299+00
32d19a5a-66d2-4a71-af69-64600463a0cd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	subscription.created	stripe	evt_28_1	2026-02-13 10:15:51.919299+00	{"demo": true, "amount_cents": 2850}	2026-02-26 03:44:51.919299+00
6a7b8efe-108b-4fff-99c1-0a3a4016cf04	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	subscription.renewed	stripe	evt_28_2	2026-02-04 10:02:51.919299+00	{"demo": true, "amount_cents": 2900}	2026-02-26 03:44:51.919299+00
1ba949bf-2724-4787-9bc4-269976fabd75	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	subscription.cancelled	stripe	evt_28_3	2026-01-26 09:49:51.919299+00	{"demo": true, "amount_cents": 2950}	2026-02-26 03:44:51.919299+00
f66f3a9c-639b-43bb-bd73-5f8eaf1ebcb3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	ticket.opened	stripe	evt_28_4	2026-01-17 09:36:51.919299+00	{"demo": true, "amount_cents": 3000}	2026-02-26 03:44:51.919299+00
06f1a5ab-c94a-4bfc-ac17-d8c334450fb2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	ticket.resolved	stripe	evt_28_5	2026-01-08 09:23:51.919299+00	{"demo": true, "amount_cents": 3050}	2026-02-26 03:44:51.919299+00
8799e1e4-c14f-4f80-86be-3dcfa085e504	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	login	stripe	evt_28_6	2025-12-30 09:10:51.919299+00	{"demo": true, "amount_cents": 3100}	2026-02-26 03:44:51.919299+00
ba51c584-36c2-4749-9180-3ab0c3e58ae1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	feature.used	stripe	evt_28_7	2025-12-21 08:57:51.919299+00	{"demo": true, "amount_cents": 3150}	2026-02-26 03:44:51.919299+00
d786101a-0cd7-4979-af6d-a0198d752586	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	payment.success	stripe	evt_28_8	2025-12-12 08:44:51.919299+00	{"demo": true, "amount_cents": 3200}	2026-02-26 03:44:51.919299+00
f27e337e-0cb2-462f-977c-53b18b45d4e0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	payment.failed	stripe	evt_28_9	2025-12-03 08:31:51.919299+00	{"demo": true, "amount_cents": 3250}	2026-02-26 03:44:51.919299+00
487f2140-4a94-43b1-88d6-45a64fcc601b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	subscription.created	stripe	evt_28_10	2025-11-24 08:18:51.919299+00	{"demo": true, "amount_cents": 3300}	2026-02-26 03:44:51.919299+00
8dab3f0e-f637-4b05-ae05-61774e838d03	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	subscription.renewed	stripe	evt_29_1	2026-02-12 09:38:51.919299+00	{"demo": true, "amount_cents": 2950}	2026-02-26 03:44:51.919299+00
6db33d90-b171-428a-b10f-cb02e9885a8f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	subscription.cancelled	stripe	evt_29_2	2026-02-03 09:25:51.919299+00	{"demo": true, "amount_cents": 3000}	2026-02-26 03:44:51.919299+00
eff3f593-2470-4769-95be-0a754290020b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	ticket.opened	stripe	evt_29_3	2026-01-25 09:12:51.919299+00	{"demo": true, "amount_cents": 3050}	2026-02-26 03:44:51.919299+00
35b03f85-3aa2-4c04-8057-7df98b5878a8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	ticket.resolved	stripe	evt_29_4	2026-01-16 08:59:51.919299+00	{"demo": true, "amount_cents": 3100}	2026-02-26 03:44:51.919299+00
7b74efab-1cae-4881-9414-98dce8daa3ce	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	login	stripe	evt_29_5	2026-01-07 08:46:51.919299+00	{"demo": true, "amount_cents": 3150}	2026-02-26 03:44:51.919299+00
947b300c-f0ee-4215-9221-c2717874e67e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	feature.used	stripe	evt_29_6	2025-12-29 08:33:51.919299+00	{"demo": true, "amount_cents": 3200}	2026-02-26 03:44:51.919299+00
e5e7ed48-09e8-4abf-92c0-bfc6535e1404	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	payment.success	stripe	evt_29_7	2025-12-20 08:20:51.919299+00	{"demo": true, "amount_cents": 3250}	2026-02-26 03:44:51.919299+00
913805ee-2e97-4fcc-9c02-f28b0bc184e4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	payment.failed	stripe	evt_29_8	2025-12-11 08:07:51.919299+00	{"demo": true, "amount_cents": 3300}	2026-02-26 03:44:51.919299+00
27c294a9-d919-4ed0-8a3c-b92b6bf2baed	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	subscription.created	stripe	evt_29_9	2025-12-02 07:54:51.919299+00	{"demo": true, "amount_cents": 3350}	2026-02-26 03:44:51.919299+00
1fb1e041-601e-47e2-aa3e-0df53526d482	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	subscription.renewed	stripe	evt_29_10	2025-11-23 07:41:51.919299+00	{"demo": true, "amount_cents": 3400}	2026-02-26 03:44:51.919299+00
af0c94bb-93c5-4c6a-b995-ebc253cfc348	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	subscription.cancelled	stripe	evt_30_1	2026-02-16 09:01:51.919299+00	{"demo": true, "amount_cents": 3050}	2026-02-26 03:44:51.919299+00
6bcef31c-680b-469a-8f35-eff38f9de009	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	ticket.opened	stripe	evt_30_2	2026-02-07 08:48:51.919299+00	{"demo": true, "amount_cents": 3100}	2026-02-26 03:44:51.919299+00
e719f145-8300-41ef-a450-a2e0fbf0120b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	ticket.resolved	stripe	evt_30_3	2026-01-29 08:35:51.919299+00	{"demo": true, "amount_cents": 3150}	2026-02-26 03:44:51.919299+00
9a39609c-6531-44b0-acdc-5be7169c72c1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	login	stripe	evt_30_4	2026-01-20 08:22:51.919299+00	{"demo": true, "amount_cents": 3200}	2026-02-26 03:44:51.919299+00
9b2a193a-7d2e-4d23-aee3-f898a6464c75	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	feature.used	stripe	evt_30_5	2026-01-11 08:09:51.919299+00	{"demo": true, "amount_cents": 3250}	2026-02-26 03:44:51.919299+00
be30c940-f8cb-48b7-bf70-04c3fec0b381	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	payment.success	stripe	evt_30_6	2026-01-02 07:56:51.919299+00	{"demo": true, "amount_cents": 3300}	2026-02-26 03:44:51.919299+00
98f45c4c-2d4d-4e3e-8b39-24a05b99a135	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	payment.failed	stripe	evt_30_7	2025-12-24 07:43:51.919299+00	{"demo": true, "amount_cents": 3350}	2026-02-26 03:44:51.919299+00
b7d10d43-549a-4173-8e0c-cbd18d2e131e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	subscription.created	stripe	evt_30_8	2025-12-15 07:30:51.919299+00	{"demo": true, "amount_cents": 3400}	2026-02-26 03:44:51.919299+00
a6eb45ec-b5b3-45c1-8973-6754daba3577	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	subscription.renewed	stripe	evt_30_9	2025-12-06 07:17:51.919299+00	{"demo": true, "amount_cents": 3450}	2026-02-26 03:44:51.919299+00
a0b77d80-cbde-45b8-b456-de1f59648f12	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	subscription.cancelled	stripe	evt_30_10	2025-11-27 07:04:51.919299+00	{"demo": true, "amount_cents": 3500}	2026-02-26 03:44:51.919299+00
83e7e2f0-a0d5-4a56-9cc8-7d058dc51775	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	ticket.opened	stripe	evt_31_1	2026-02-15 08:24:51.919299+00	{"demo": true, "amount_cents": 3150}	2026-02-26 03:44:51.919299+00
77a43667-b7f6-407b-b94f-32b4407aa20c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	ticket.resolved	stripe	evt_31_2	2026-02-06 08:11:51.919299+00	{"demo": true, "amount_cents": 3200}	2026-02-26 03:44:51.919299+00
45aa62c0-d936-4229-8ed1-c1d09f1349b2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	login	stripe	evt_31_3	2026-01-28 07:58:51.919299+00	{"demo": true, "amount_cents": 3250}	2026-02-26 03:44:51.919299+00
ef61f2c4-69d6-420e-a1fe-bd0c888658a1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	feature.used	stripe	evt_31_4	2026-01-19 07:45:51.919299+00	{"demo": true, "amount_cents": 3300}	2026-02-26 03:44:51.919299+00
43323d07-dff3-4493-99a6-e4cbc1a114ab	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	payment.success	stripe	evt_31_5	2026-01-10 07:32:51.919299+00	{"demo": true, "amount_cents": 3350}	2026-02-26 03:44:51.919299+00
340b1e98-60c9-4865-a026-06a7bbf7babe	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	payment.failed	stripe	evt_31_6	2026-01-01 07:19:51.919299+00	{"demo": true, "amount_cents": 3400}	2026-02-26 03:44:51.919299+00
13ae0bec-d548-4d82-8111-ae8449fa4d1e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	subscription.created	stripe	evt_31_7	2025-12-23 07:06:51.919299+00	{"demo": true, "amount_cents": 3450}	2026-02-26 03:44:51.919299+00
12e077ac-626e-4e03-ac6e-2e319d24cd21	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	subscription.renewed	stripe	evt_31_8	2025-12-14 06:53:51.919299+00	{"demo": true, "amount_cents": 3500}	2026-02-26 03:44:51.919299+00
47de4328-9f24-4c8d-a5ca-98278d77cb37	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	subscription.cancelled	stripe	evt_31_9	2025-12-05 06:40:51.919299+00	{"demo": true, "amount_cents": 3550}	2026-02-26 03:44:51.919299+00
fe3b96db-ffe0-476b-95c4-7469442df7a2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	ticket.opened	stripe	evt_31_10	2025-11-26 06:27:51.919299+00	{"demo": true, "amount_cents": 3600}	2026-02-26 03:44:51.919299+00
31ad20f0-a39f-4db5-8cca-16efcbee795e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	ticket.resolved	stripe	evt_32_1	2026-02-14 07:47:51.919299+00	{"demo": true, "amount_cents": 3250}	2026-02-26 03:44:51.919299+00
1d64fbbc-7bb2-454d-9b19-08a763f22171	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	login	stripe	evt_32_2	2026-02-05 07:34:51.919299+00	{"demo": true, "amount_cents": 3300}	2026-02-26 03:44:51.919299+00
063b0e81-b704-4c21-8e98-46f498f3e0ca	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	feature.used	stripe	evt_32_3	2026-01-27 07:21:51.919299+00	{"demo": true, "amount_cents": 3350}	2026-02-26 03:44:51.919299+00
263fd10d-7aa9-4326-be2d-4fd50044281f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	payment.success	stripe	evt_32_4	2026-01-18 07:08:51.919299+00	{"demo": true, "amount_cents": 3400}	2026-02-26 03:44:51.919299+00
5c964873-361e-4d4f-80df-d34d6e71f2db	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	payment.failed	stripe	evt_32_5	2026-01-09 06:55:51.919299+00	{"demo": true, "amount_cents": 3450}	2026-02-26 03:44:51.919299+00
c84fec8e-75b5-4c43-a00a-7c56145c8c7b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	subscription.created	stripe	evt_32_6	2025-12-31 06:42:51.919299+00	{"demo": true, "amount_cents": 3500}	2026-02-26 03:44:51.919299+00
b5d04793-2cb4-4b5c-8ebf-e297082d869d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	subscription.renewed	stripe	evt_32_7	2025-12-22 06:29:51.919299+00	{"demo": true, "amount_cents": 3550}	2026-02-26 03:44:51.919299+00
8cceee8d-2ee7-4ff4-94c5-b36cb462a022	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	subscription.cancelled	stripe	evt_32_8	2025-12-13 06:16:51.919299+00	{"demo": true, "amount_cents": 3600}	2026-02-26 03:44:51.919299+00
7d87b1a3-215c-4879-97f6-11a6d182d346	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	ticket.opened	stripe	evt_32_9	2025-12-04 06:03:51.919299+00	{"demo": true, "amount_cents": 3650}	2026-02-26 03:44:51.919299+00
477f679c-da29-4c80-907a-40074dff360b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	ticket.resolved	stripe	evt_32_10	2025-11-25 05:50:51.919299+00	{"demo": true, "amount_cents": 3700}	2026-02-26 03:44:51.919299+00
f121762c-6966-425e-a950-79ac79bc7efa	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	login	stripe	evt_33_1	2026-02-13 07:10:51.919299+00	{"demo": true, "amount_cents": 3350}	2026-02-26 03:44:51.919299+00
2b5d8a00-4309-4038-a551-2b4fb0d6569a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	feature.used	stripe	evt_33_2	2026-02-04 06:57:51.919299+00	{"demo": true, "amount_cents": 3400}	2026-02-26 03:44:51.919299+00
53c9d0ef-7577-430b-8f76-a9dadb75c913	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	payment.success	stripe	evt_33_3	2026-01-26 06:44:51.919299+00	{"demo": true, "amount_cents": 3450}	2026-02-26 03:44:51.919299+00
ca72fac1-dd21-4b92-8ff5-afd93b84b565	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	payment.failed	stripe	evt_33_4	2026-01-17 06:31:51.919299+00	{"demo": true, "amount_cents": 3500}	2026-02-26 03:44:51.919299+00
8fa32532-a1c7-408a-bc05-12ce4f8b8932	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	subscription.created	stripe	evt_33_5	2026-01-08 06:18:51.919299+00	{"demo": true, "amount_cents": 3550}	2026-02-26 03:44:51.919299+00
6cbcc07a-b7ec-48c8-aeb5-52ee9dc460f8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	subscription.renewed	stripe	evt_33_6	2025-12-30 06:05:51.919299+00	{"demo": true, "amount_cents": 3600}	2026-02-26 03:44:51.919299+00
424c75ca-1e40-4fde-8319-db6da3ed6cad	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	subscription.cancelled	stripe	evt_33_7	2025-12-21 05:52:51.919299+00	{"demo": true, "amount_cents": 3650}	2026-02-26 03:44:51.919299+00
bd4fc82e-475e-423f-a9af-3cbcd841bb6b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	ticket.opened	stripe	evt_33_8	2025-12-12 05:39:51.919299+00	{"demo": true, "amount_cents": 3700}	2026-02-26 03:44:51.919299+00
af69b38d-5ac9-479b-b77b-b6ab70ced32f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	ticket.resolved	stripe	evt_33_9	2025-12-03 05:26:51.919299+00	{"demo": true, "amount_cents": 3750}	2026-02-26 03:44:51.919299+00
9899f1cd-1427-46be-8758-e6b83b343079	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	login	stripe	evt_33_10	2025-11-24 05:13:51.919299+00	{"demo": true, "amount_cents": 3800}	2026-02-26 03:44:51.919299+00
32b6fac1-7af3-458d-ad6a-8432103f16c9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	feature.used	stripe	evt_34_1	2026-02-12 06:33:51.919299+00	{"demo": true, "amount_cents": 3450}	2026-02-26 03:44:51.919299+00
275c47d9-1ec9-4811-b284-5dce14d93dea	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	payment.success	stripe	evt_34_2	2026-02-03 06:20:51.919299+00	{"demo": true, "amount_cents": 3500}	2026-02-26 03:44:51.919299+00
3ab82644-8733-412f-bffa-5c7eaebc2cd5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	payment.failed	stripe	evt_34_3	2026-01-25 06:07:51.919299+00	{"demo": true, "amount_cents": 3550}	2026-02-26 03:44:51.919299+00
3c06f08d-fe3b-42f8-ac08-3d130936d241	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	subscription.created	stripe	evt_34_4	2026-01-16 05:54:51.919299+00	{"demo": true, "amount_cents": 3600}	2026-02-26 03:44:51.919299+00
d9b231a4-8ee0-455a-b8a8-4700868d34e5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	subscription.renewed	stripe	evt_34_5	2026-01-07 05:41:51.919299+00	{"demo": true, "amount_cents": 3650}	2026-02-26 03:44:51.919299+00
f07a9f1b-b80a-4176-ba0c-6a453e47ab55	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	subscription.cancelled	stripe	evt_34_6	2025-12-29 05:28:51.919299+00	{"demo": true, "amount_cents": 3700}	2026-02-26 03:44:51.919299+00
671aff93-cb90-4fc9-a4ce-f0f9c5109fcf	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	ticket.opened	stripe	evt_34_7	2025-12-20 05:15:51.919299+00	{"demo": true, "amount_cents": 3750}	2026-02-26 03:44:51.919299+00
16d72d33-baf5-42ca-baee-5815af813593	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	ticket.resolved	stripe	evt_34_8	2025-12-11 05:02:51.919299+00	{"demo": true, "amount_cents": 3800}	2026-02-26 03:44:51.919299+00
5393505d-07d1-4586-b43e-ddd02993773b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	login	stripe	evt_34_9	2025-12-02 04:49:51.919299+00	{"demo": true, "amount_cents": 3850}	2026-02-26 03:44:51.919299+00
1826b235-efc1-4cee-a79c-33080c874880	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	feature.used	stripe	evt_34_10	2025-11-23 04:36:51.919299+00	{"demo": true, "amount_cents": 3900}	2026-02-26 03:44:51.919299+00
ac1b5af3-8e54-4646-8144-89336e5c4648	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	payment.success	stripe	evt_35_1	2026-02-16 05:56:51.919299+00	{"demo": true, "amount_cents": 3550}	2026-02-26 03:44:51.919299+00
714dfc43-daf5-4215-8277-3fa3d917bd64	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	payment.failed	stripe	evt_35_2	2026-02-07 05:43:51.919299+00	{"demo": true, "amount_cents": 3600}	2026-02-26 03:44:51.919299+00
84a1ec75-0f09-4538-892c-bcbaee5ff9b6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	subscription.created	stripe	evt_35_3	2026-01-29 05:30:51.919299+00	{"demo": true, "amount_cents": 3650}	2026-02-26 03:44:51.919299+00
8a2a15ec-4a57-458d-99f3-48b1b5bd9311	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	subscription.renewed	stripe	evt_35_4	2026-01-20 05:17:51.919299+00	{"demo": true, "amount_cents": 3700}	2026-02-26 03:44:51.919299+00
577e2e58-31f4-4e65-8f0d-738d39c965bc	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	subscription.cancelled	stripe	evt_35_5	2026-01-11 05:04:51.919299+00	{"demo": true, "amount_cents": 3750}	2026-02-26 03:44:51.919299+00
ba964962-39ed-4600-81c9-4f32a970c49b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	ticket.opened	stripe	evt_35_6	2026-01-02 04:51:51.919299+00	{"demo": true, "amount_cents": 3800}	2026-02-26 03:44:51.919299+00
b67146b6-da82-47ad-b72b-9010d484d0c2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	ticket.resolved	stripe	evt_35_7	2025-12-24 04:38:51.919299+00	{"demo": true, "amount_cents": 3850}	2026-02-26 03:44:51.919299+00
667e6a55-007e-400f-9448-24364722017f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	login	stripe	evt_35_8	2025-12-15 04:25:51.919299+00	{"demo": true, "amount_cents": 3900}	2026-02-26 03:44:51.919299+00
291b9a22-bd34-434e-ab91-ee778bae0a10	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	feature.used	stripe	evt_35_9	2025-12-06 04:12:51.919299+00	{"demo": true, "amount_cents": 3950}	2026-02-26 03:44:51.919299+00
bfaf3319-222f-4ded-99c6-e98b2735a824	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	payment.success	stripe	evt_35_10	2025-11-27 03:59:51.919299+00	{"demo": true, "amount_cents": 4000}	2026-02-26 03:44:51.919299+00
c170acfc-972a-45c8-962f-0847d4a7aea0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	payment.failed	stripe	evt_36_1	2026-02-15 05:19:51.919299+00	{"demo": true, "amount_cents": 3650}	2026-02-26 03:44:51.919299+00
ee7e5e81-b61f-48e1-9c1d-d5b37cf33f3e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	subscription.created	stripe	evt_36_2	2026-02-06 05:06:51.919299+00	{"demo": true, "amount_cents": 3700}	2026-02-26 03:44:51.919299+00
1e9779a6-07a9-4b44-b59f-c3833e0b2752	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	subscription.renewed	stripe	evt_36_3	2026-01-28 04:53:51.919299+00	{"demo": true, "amount_cents": 3750}	2026-02-26 03:44:51.919299+00
64640769-da87-4ff7-adb4-ed4daa99cee0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	subscription.cancelled	stripe	evt_36_4	2026-01-19 04:40:51.919299+00	{"demo": true, "amount_cents": 3800}	2026-02-26 03:44:51.919299+00
0cc1d45d-def9-41da-908f-1637c280edab	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	ticket.opened	stripe	evt_36_5	2026-01-10 04:27:51.919299+00	{"demo": true, "amount_cents": 3850}	2026-02-26 03:44:51.919299+00
e8735d70-663f-46d9-ac1e-582ecf4148fd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	ticket.resolved	stripe	evt_36_6	2026-01-01 04:14:51.919299+00	{"demo": true, "amount_cents": 3900}	2026-02-26 03:44:51.919299+00
9bc6d45a-5775-44e9-92f8-e6fa9f40ae98	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	login	stripe	evt_36_7	2025-12-23 04:01:51.919299+00	{"demo": true, "amount_cents": 3950}	2026-02-26 03:44:51.919299+00
af4144ab-4913-43d1-bb26-cef561159579	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	feature.used	stripe	evt_36_8	2025-12-14 03:48:51.919299+00	{"demo": true, "amount_cents": 4000}	2026-02-26 03:44:51.919299+00
a6dce68d-5e83-4a92-913b-5dcb2b00a4d3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	payment.success	stripe	evt_36_9	2025-12-06 03:35:51.919299+00	{"demo": true, "amount_cents": 4050}	2026-02-26 03:44:51.919299+00
eba2f7d8-3f05-4357-877e-3e2c2f9ff856	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	payment.failed	stripe	evt_36_10	2025-11-27 03:22:51.919299+00	{"demo": true, "amount_cents": 4100}	2026-02-26 03:44:51.919299+00
e05a165f-1337-441b-a5c4-e4b0be46a01e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	subscription.created	stripe	evt_37_1	2026-02-14 04:42:51.919299+00	{"demo": true, "amount_cents": 3750}	2026-02-26 03:44:51.919299+00
5b798499-b106-421b-affb-43c6a43b4937	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	subscription.renewed	stripe	evt_37_2	2026-02-05 04:29:51.919299+00	{"demo": true, "amount_cents": 3800}	2026-02-26 03:44:51.919299+00
9c4cf47c-05ae-496c-909e-e2b23deb5c63	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	subscription.cancelled	stripe	evt_37_3	2026-01-27 04:16:51.919299+00	{"demo": true, "amount_cents": 3850}	2026-02-26 03:44:51.919299+00
08b0fcea-95cd-4ce1-96bf-2adbfc97bbf4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	ticket.opened	stripe	evt_37_4	2026-01-18 04:03:51.919299+00	{"demo": true, "amount_cents": 3900}	2026-02-26 03:44:51.919299+00
fd2837fa-6a26-45b6-b130-f3628bac82dd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	ticket.resolved	stripe	evt_37_5	2026-01-09 03:50:51.919299+00	{"demo": true, "amount_cents": 3950}	2026-02-26 03:44:51.919299+00
6fb939dc-5585-4711-936b-2c657277f9f9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	login	stripe	evt_37_6	2026-01-01 03:37:51.919299+00	{"demo": true, "amount_cents": 4000}	2026-02-26 03:44:51.919299+00
75b5e307-ef7d-4375-9386-97c6ccb1ab3a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	feature.used	stripe	evt_37_7	2025-12-23 03:24:51.919299+00	{"demo": true, "amount_cents": 4050}	2026-02-26 03:44:51.919299+00
56417ccc-2f4b-4a7f-9f50-c0d6cb8c2e9d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	payment.success	stripe	evt_37_8	2025-12-14 03:11:51.919299+00	{"demo": true, "amount_cents": 4100}	2026-02-26 03:44:51.919299+00
977a006c-87fc-4636-ad7c-d18275a723a5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	payment.failed	stripe	evt_37_9	2025-12-05 02:58:51.919299+00	{"demo": true, "amount_cents": 4150}	2026-02-26 03:44:51.919299+00
4cedf8f2-a80c-4967-ae17-ad623577af31	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	subscription.created	stripe	evt_37_10	2025-11-26 02:45:51.919299+00	{"demo": true, "amount_cents": 4200}	2026-02-26 03:44:51.919299+00
e3bcfcd0-2b17-4ea0-821b-5d322a207006	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	subscription.renewed	stripe	evt_38_1	2026-02-13 04:05:51.919299+00	{"demo": true, "amount_cents": 3850}	2026-02-26 03:44:51.919299+00
c32831b6-26a5-4d60-9919-88bf21e643f5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	subscription.cancelled	stripe	evt_38_2	2026-02-04 03:52:51.919299+00	{"demo": true, "amount_cents": 3900}	2026-02-26 03:44:51.919299+00
bb7605ed-6749-4257-9895-d75896083861	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	ticket.opened	stripe	evt_38_3	2026-01-27 03:39:51.919299+00	{"demo": true, "amount_cents": 3950}	2026-02-26 03:44:51.919299+00
4484b188-7ea5-46e9-b16a-67d903634388	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	ticket.resolved	stripe	evt_38_4	2026-01-18 03:26:51.919299+00	{"demo": true, "amount_cents": 4000}	2026-02-26 03:44:51.919299+00
45c97cfb-8ac8-4880-a3bd-ab649abd4277	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	login	stripe	evt_38_5	2026-01-09 03:13:51.919299+00	{"demo": true, "amount_cents": 4050}	2026-02-26 03:44:51.919299+00
c2c57edf-fa00-490c-9948-1440e8f7cffb	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	feature.used	stripe	evt_38_6	2025-12-31 03:00:51.919299+00	{"demo": true, "amount_cents": 4100}	2026-02-26 03:44:51.919299+00
ed33e573-6458-4b54-a0c6-7363dafb7dd8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	payment.success	stripe	evt_38_7	2025-12-22 02:47:51.919299+00	{"demo": true, "amount_cents": 4150}	2026-02-26 03:44:51.919299+00
6e7db24b-e8d2-4efe-ab5c-1bd6c236d19d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	payment.failed	stripe	evt_38_8	2025-12-13 02:34:51.919299+00	{"demo": true, "amount_cents": 4200}	2026-02-26 03:44:51.919299+00
20af1547-3276-4525-85a8-83738b8feb22	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	subscription.created	stripe	evt_38_9	2025-12-04 02:21:51.919299+00	{"demo": true, "amount_cents": 4250}	2026-02-26 03:44:51.919299+00
b2a24d88-5b08-4792-b591-514b97b494c1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	subscription.renewed	stripe	evt_38_10	2025-11-25 02:08:51.919299+00	{"demo": true, "amount_cents": 4300}	2026-02-26 03:44:51.919299+00
3698d8b3-3fb8-4796-8e9f-f6e3d2194d16	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	subscription.cancelled	stripe	evt_39_1	2026-02-13 03:28:51.919299+00	{"demo": true, "amount_cents": 3950}	2026-02-26 03:44:51.919299+00
ff2c9593-5ef5-43d6-a3cf-607730f22f4d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	ticket.opened	stripe	evt_39_2	2026-02-04 03:15:51.919299+00	{"demo": true, "amount_cents": 4000}	2026-02-26 03:44:51.919299+00
7a80c68b-734c-4d08-a0bb-57ac34b24f4d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	ticket.resolved	stripe	evt_39_3	2026-01-26 03:02:51.919299+00	{"demo": true, "amount_cents": 4050}	2026-02-26 03:44:51.919299+00
7d1613a8-253e-425d-b342-2a3b1b695a13	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	login	stripe	evt_39_4	2026-01-17 02:49:51.919299+00	{"demo": true, "amount_cents": 4100}	2026-02-26 03:44:51.919299+00
c7fbd71c-fd9c-4d40-8884-9843dee94b73	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	feature.used	stripe	evt_39_5	2026-01-08 02:36:51.919299+00	{"demo": true, "amount_cents": 4150}	2026-02-26 03:44:51.919299+00
5fdee200-5f59-44d4-919a-8237c2a0a8db	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	payment.success	stripe	evt_39_6	2025-12-30 02:23:51.919299+00	{"demo": true, "amount_cents": 4200}	2026-02-26 03:44:51.919299+00
fd981185-2c5c-4378-b2fc-c6e998d2161f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	payment.failed	stripe	evt_39_7	2025-12-21 02:10:51.919299+00	{"demo": true, "amount_cents": 4250}	2026-02-26 03:44:51.919299+00
fffb3ed3-204a-48a6-af68-12b0da124a2a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	subscription.created	stripe	evt_39_8	2025-12-12 01:57:51.919299+00	{"demo": true, "amount_cents": 4300}	2026-02-26 03:44:51.919299+00
5462b047-7cd4-4ff6-8b82-4c582aa7bd4d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	subscription.renewed	stripe	evt_39_9	2025-12-03 01:44:51.919299+00	{"demo": true, "amount_cents": 4350}	2026-02-26 03:44:51.919299+00
7483a9e8-87d3-4fd1-8674-8f7a465aa2be	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	subscription.cancelled	stripe	evt_39_10	2025-11-24 01:31:51.919299+00	{"demo": true, "amount_cents": 4400}	2026-02-26 03:44:51.919299+00
c0a8abf2-6050-4e80-87ab-fc36def2d27a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	ticket.opened	stripe	evt_40_1	2026-02-17 02:51:51.919299+00	{"demo": true, "amount_cents": 4050}	2026-02-26 03:44:51.919299+00
e608351e-cddc-4188-aca1-2dd444ff8230	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	ticket.resolved	stripe	evt_40_2	2026-02-08 02:38:51.919299+00	{"demo": true, "amount_cents": 4100}	2026-02-26 03:44:51.919299+00
18770bb6-3a1a-4043-be9d-eb1d1eca5168	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	login	stripe	evt_40_3	2026-01-30 02:25:51.919299+00	{"demo": true, "amount_cents": 4150}	2026-02-26 03:44:51.919299+00
f2dc2347-311e-458f-9d49-02821b58ebd3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	feature.used	stripe	evt_40_4	2026-01-21 02:12:51.919299+00	{"demo": true, "amount_cents": 4200}	2026-02-26 03:44:51.919299+00
79d0dc61-5c96-46c9-834c-512b28bd37a4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	payment.success	stripe	evt_40_5	2026-01-12 01:59:51.919299+00	{"demo": true, "amount_cents": 4250}	2026-02-26 03:44:51.919299+00
3702eb30-0f0a-4e67-830c-4d23408fc96c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	payment.failed	stripe	evt_40_6	2026-01-03 01:46:51.919299+00	{"demo": true, "amount_cents": 4300}	2026-02-26 03:44:51.919299+00
aefe5cf4-6085-4521-a211-e5bbc96af3a7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	subscription.created	stripe	evt_40_7	2025-12-25 01:33:51.919299+00	{"demo": true, "amount_cents": 4350}	2026-02-26 03:44:51.919299+00
2d19fe1c-cd98-497b-bfc9-e1273041b7aa	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	subscription.renewed	stripe	evt_40_8	2025-12-16 01:20:51.919299+00	{"demo": true, "amount_cents": 4400}	2026-02-26 03:44:51.919299+00
037478bc-9633-4d8f-8184-d192d8d1c878	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	subscription.cancelled	stripe	evt_40_9	2025-12-07 01:07:51.919299+00	{"demo": true, "amount_cents": 4450}	2026-02-26 03:44:51.919299+00
999ca244-5594-4f21-9a0c-2435f8322f5a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	ticket.opened	stripe	evt_40_10	2025-11-28 00:54:51.919299+00	{"demo": true, "amount_cents": 4500}	2026-02-26 03:44:51.919299+00
aebbbad5-1f1a-447f-9821-a5cf5fd44ad5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	ticket.resolved	stripe	evt_41_1	2026-02-16 02:14:51.919299+00	{"demo": true, "amount_cents": 4150}	2026-02-26 03:44:51.919299+00
939db5c1-e050-4df0-9a01-94bf9ee27056	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	login	stripe	evt_41_2	2026-02-07 02:01:51.919299+00	{"demo": true, "amount_cents": 4200}	2026-02-26 03:44:51.919299+00
470f391a-b7be-43cb-9298-e5e4c1f99533	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	feature.used	stripe	evt_41_3	2026-01-29 01:48:51.919299+00	{"demo": true, "amount_cents": 4250}	2026-02-26 03:44:51.919299+00
d07b06f5-dfd4-4fe2-b41a-6e130eb92c4a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	payment.success	stripe	evt_41_4	2026-01-20 01:35:51.919299+00	{"demo": true, "amount_cents": 4300}	2026-02-26 03:44:51.919299+00
6d5b0033-0606-4012-b8e1-72cabcf2ba6e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	payment.failed	stripe	evt_41_5	2026-01-11 01:22:51.919299+00	{"demo": true, "amount_cents": 4350}	2026-02-26 03:44:51.919299+00
ded6bc48-b9e7-44d7-a351-96265c1652fa	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	subscription.created	stripe	evt_41_6	2026-01-02 01:09:51.919299+00	{"demo": true, "amount_cents": 4400}	2026-02-26 03:44:51.919299+00
a0aeb166-8403-4555-9b0a-a3dca2e4c798	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	subscription.renewed	stripe	evt_41_7	2025-12-24 00:56:51.919299+00	{"demo": true, "amount_cents": 4450}	2026-02-26 03:44:51.919299+00
baf42deb-7551-4b3e-bb03-e4fa4c7642a8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	subscription.cancelled	stripe	evt_41_8	2025-12-15 00:43:51.919299+00	{"demo": true, "amount_cents": 4500}	2026-02-26 03:44:51.919299+00
a1cf5b0c-c7c1-49a9-b777-aee936433037	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	ticket.opened	stripe	evt_41_9	2025-12-06 00:30:51.919299+00	{"demo": true, "amount_cents": 4550}	2026-02-26 03:44:51.919299+00
93de72e9-4752-4a6e-b078-e5fbac36128a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	ticket.resolved	stripe	evt_41_10	2025-11-27 00:17:51.919299+00	{"demo": true, "amount_cents": 4600}	2026-02-26 03:44:51.919299+00
0169b089-8b6a-43fb-954a-c869a1aa82a7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	login	stripe	evt_42_1	2026-02-15 01:37:51.919299+00	{"demo": true, "amount_cents": 4250}	2026-02-26 03:44:51.919299+00
95e5364c-4441-4df8-909f-e59114a81d26	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	feature.used	stripe	evt_42_2	2026-02-06 01:24:51.919299+00	{"demo": true, "amount_cents": 4300}	2026-02-26 03:44:51.919299+00
bc8ce9f7-d711-40e7-afef-9f3cda014fb7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	payment.success	stripe	evt_42_3	2026-01-28 01:11:51.919299+00	{"demo": true, "amount_cents": 4350}	2026-02-26 03:44:51.919299+00
86be2b55-30c1-41e4-be6a-f6aad2ea0bb5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	payment.failed	stripe	evt_42_4	2026-01-19 00:58:51.919299+00	{"demo": true, "amount_cents": 4400}	2026-02-26 03:44:51.919299+00
aa618b6f-fd84-4633-bd98-c85b263a59d2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	subscription.created	stripe	evt_42_5	2026-01-10 00:45:51.919299+00	{"demo": true, "amount_cents": 4450}	2026-02-26 03:44:51.919299+00
5c6a2f98-6635-435c-a823-40d5ef5958a1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	subscription.renewed	stripe	evt_42_6	2026-01-01 00:32:51.919299+00	{"demo": true, "amount_cents": 4500}	2026-02-26 03:44:51.919299+00
d7a8c313-3ed2-4582-ac6f-6c5b35285dd5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	subscription.cancelled	stripe	evt_42_7	2025-12-23 00:19:51.919299+00	{"demo": true, "amount_cents": 4550}	2026-02-26 03:44:51.919299+00
af77b8fe-ce2f-4330-9e11-97ccb93444f1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	ticket.opened	stripe	evt_42_8	2025-12-14 00:06:51.919299+00	{"demo": true, "amount_cents": 4600}	2026-02-26 03:44:51.919299+00
9481179c-4442-469a-b198-cfd5fb17e6a3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	ticket.resolved	stripe	evt_42_9	2025-12-04 23:53:51.919299+00	{"demo": true, "amount_cents": 4650}	2026-02-26 03:44:51.919299+00
a5b9c21b-89b5-495d-8b79-7a8674246df5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	login	stripe	evt_42_10	2025-11-25 23:40:51.919299+00	{"demo": true, "amount_cents": 4700}	2026-02-26 03:44:51.919299+00
7c87ad70-aad4-469e-abb2-bc5c4b963b72	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	feature.used	stripe	evt_43_1	2026-02-14 01:00:51.919299+00	{"demo": true, "amount_cents": 4350}	2026-02-26 03:44:51.919299+00
6d109993-7f92-48d4-bb88-2c310c54c789	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	payment.success	stripe	evt_43_2	2026-02-05 00:47:51.919299+00	{"demo": true, "amount_cents": 4400}	2026-02-26 03:44:51.919299+00
c339ba2a-b028-4594-bae3-f9ee89ebe353	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	payment.failed	stripe	evt_43_3	2026-01-27 00:34:51.919299+00	{"demo": true, "amount_cents": 4450}	2026-02-26 03:44:51.919299+00
d30dbe26-f074-41fc-9681-cf42fe025a79	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	subscription.created	stripe	evt_43_4	2026-01-18 00:21:51.919299+00	{"demo": true, "amount_cents": 4500}	2026-02-26 03:44:51.919299+00
442d5f85-e79a-47e5-bd03-996a2e1b1a81	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	subscription.renewed	stripe	evt_43_5	2026-01-09 00:08:51.919299+00	{"demo": true, "amount_cents": 4550}	2026-02-26 03:44:51.919299+00
cec491dd-b366-4bf9-a989-f9d3102a1734	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	subscription.cancelled	stripe	evt_43_6	2025-12-30 23:55:51.919299+00	{"demo": true, "amount_cents": 4600}	2026-02-26 03:44:51.919299+00
29b5110c-35b6-49a1-9760-920f71558984	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	ticket.opened	stripe	evt_43_7	2025-12-21 23:42:51.919299+00	{"demo": true, "amount_cents": 4650}	2026-02-26 03:44:51.919299+00
64a3ed6c-6d18-41f1-b918-ad14c8c9947a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	ticket.resolved	stripe	evt_43_8	2025-12-12 23:29:51.919299+00	{"demo": true, "amount_cents": 4700}	2026-02-26 03:44:51.919299+00
07a44bc1-c294-4c4a-823e-b780499b7f44	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	login	stripe	evt_43_9	2025-12-03 23:16:51.919299+00	{"demo": true, "amount_cents": 4750}	2026-02-26 03:44:51.919299+00
dcd55669-28ac-43d4-bae6-2509dee1d14b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	feature.used	stripe	evt_43_10	2025-11-24 23:03:51.919299+00	{"demo": true, "amount_cents": 4800}	2026-02-26 03:44:51.919299+00
f86bc823-9566-4c15-b640-e6cc586d2588	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	payment.success	stripe	evt_44_1	2026-02-13 00:23:51.919299+00	{"demo": true, "amount_cents": 4450}	2026-02-26 03:44:51.919299+00
c8240173-2066-44fd-92db-891573299fcf	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	payment.failed	stripe	evt_44_2	2026-02-04 00:10:51.919299+00	{"demo": true, "amount_cents": 4500}	2026-02-26 03:44:51.919299+00
6044cea4-078a-4c5f-84c7-99e190f5e7c4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	subscription.created	stripe	evt_44_3	2026-01-25 23:57:51.919299+00	{"demo": true, "amount_cents": 4550}	2026-02-26 03:44:51.919299+00
a10d8e49-80df-41fe-9b6c-47e929750e97	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	subscription.renewed	stripe	evt_44_4	2026-01-16 23:44:51.919299+00	{"demo": true, "amount_cents": 4600}	2026-02-26 03:44:51.919299+00
2d382fe3-b8be-4534-87d4-1823df29e2ce	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	subscription.cancelled	stripe	evt_44_5	2026-01-07 23:31:51.919299+00	{"demo": true, "amount_cents": 4650}	2026-02-26 03:44:51.919299+00
f75b1cf4-a8c5-4f78-87ec-0b014f08b0ff	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	ticket.opened	stripe	evt_44_6	2025-12-29 23:18:51.919299+00	{"demo": true, "amount_cents": 4700}	2026-02-26 03:44:51.919299+00
3eef2b9a-146c-4844-8f3f-a771d43e2597	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	ticket.resolved	stripe	evt_44_7	2025-12-20 23:05:51.919299+00	{"demo": true, "amount_cents": 4750}	2026-02-26 03:44:51.919299+00
768c6daa-c584-4b09-a411-3b96e6c193d0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	login	stripe	evt_44_8	2025-12-11 22:52:51.919299+00	{"demo": true, "amount_cents": 4800}	2026-02-26 03:44:51.919299+00
ebdd67e5-e906-4dbe-89e1-703c127ba94e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	feature.used	stripe	evt_44_9	2025-12-02 22:39:51.919299+00	{"demo": true, "amount_cents": 4850}	2026-02-26 03:44:51.919299+00
2174f789-d603-40ae-b1c2-28fe03df3e67	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	payment.success	stripe	evt_44_10	2025-11-23 22:26:51.919299+00	{"demo": true, "amount_cents": 4900}	2026-02-26 03:44:51.919299+00
e374798e-fcbd-499a-ac02-b386a6e07097	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	payment.failed	stripe	evt_45_1	2026-02-16 23:46:51.919299+00	{"demo": true, "amount_cents": 4550}	2026-02-26 03:44:51.919299+00
f6f5e168-9d76-489a-aea2-45dd05475390	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	subscription.created	stripe	evt_45_2	2026-02-07 23:33:51.919299+00	{"demo": true, "amount_cents": 4600}	2026-02-26 03:44:51.919299+00
69702152-7257-489d-8475-7ed031dbccde	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	subscription.renewed	stripe	evt_45_3	2026-01-29 23:20:51.919299+00	{"demo": true, "amount_cents": 4650}	2026-02-26 03:44:51.919299+00
b4613255-50f5-46b0-8052-e035041a56a9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	subscription.cancelled	stripe	evt_45_4	2026-01-20 23:07:51.919299+00	{"demo": true, "amount_cents": 4700}	2026-02-26 03:44:51.919299+00
a8d57fea-9d3a-4126-b54c-4219cd9b2a05	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	ticket.opened	stripe	evt_45_5	2026-01-11 22:54:51.919299+00	{"demo": true, "amount_cents": 4750}	2026-02-26 03:44:51.919299+00
252270c1-5aab-4992-a2b7-9d406872d26b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	ticket.resolved	stripe	evt_45_6	2026-01-02 22:41:51.919299+00	{"demo": true, "amount_cents": 4800}	2026-02-26 03:44:51.919299+00
881d719c-6012-4e8b-9200-96d552c63366	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	login	stripe	evt_45_7	2025-12-24 22:28:51.919299+00	{"demo": true, "amount_cents": 4850}	2026-02-26 03:44:51.919299+00
a9eaadce-98f4-47cd-82f5-f3e6d9c41f0a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	feature.used	stripe	evt_45_8	2025-12-15 22:15:51.919299+00	{"demo": true, "amount_cents": 4900}	2026-02-26 03:44:51.919299+00
ee426cbc-8806-41f6-8867-c63f1fe09ee4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	payment.success	stripe	evt_45_9	2025-12-06 22:02:51.919299+00	{"demo": true, "amount_cents": 4950}	2026-02-26 03:44:51.919299+00
4fca95c3-21fe-450d-bb51-b310f2f1f7dd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	payment.failed	stripe	evt_45_10	2025-11-27 21:49:51.919299+00	{"demo": true, "amount_cents": 5000}	2026-02-26 03:44:51.919299+00
ab1bc55b-be48-4ac4-af3a-9286f23ed8ba	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	subscription.created	stripe	evt_46_1	2026-02-15 23:09:51.919299+00	{"demo": true, "amount_cents": 4650}	2026-02-26 03:44:51.919299+00
d8b771a8-46d9-4120-af90-9763c44f20dd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	subscription.renewed	stripe	evt_46_2	2026-02-06 22:56:51.919299+00	{"demo": true, "amount_cents": 4700}	2026-02-26 03:44:51.919299+00
0ea7f2d3-336b-4d2a-81a5-874090112ff9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	subscription.cancelled	stripe	evt_46_3	2026-01-28 22:43:51.919299+00	{"demo": true, "amount_cents": 4750}	2026-02-26 03:44:51.919299+00
03b2c662-3ed2-46fc-bdb4-f0c3be6a2419	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	ticket.opened	stripe	evt_46_4	2026-01-19 22:30:51.919299+00	{"demo": true, "amount_cents": 4800}	2026-02-26 03:44:51.919299+00
cb41ed57-578f-4121-8682-32bdd6eb4ce7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	ticket.resolved	stripe	evt_46_5	2026-01-10 22:17:51.919299+00	{"demo": true, "amount_cents": 4850}	2026-02-26 03:44:51.919299+00
167a4eea-4ac1-4886-a5da-df831b432f8c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	login	stripe	evt_46_6	2026-01-01 22:04:51.919299+00	{"demo": true, "amount_cents": 4900}	2026-02-26 03:44:51.919299+00
2ea4cfd6-d5f3-44bc-9d9a-252f81fce89c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	feature.used	stripe	evt_46_7	2025-12-23 21:51:51.919299+00	{"demo": true, "amount_cents": 4950}	2026-02-26 03:44:51.919299+00
736a512d-c481-42de-ae23-544c938a07dd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	payment.success	stripe	evt_46_8	2025-12-14 21:38:51.919299+00	{"demo": true, "amount_cents": 5000}	2026-02-26 03:44:51.919299+00
fc6a3f64-fc27-42e8-b5c0-958cb078a065	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	payment.failed	stripe	evt_46_9	2025-12-05 21:25:51.919299+00	{"demo": true, "amount_cents": 5050}	2026-02-26 03:44:51.919299+00
f7df816e-918c-4f22-b735-3eacefb4af18	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	subscription.created	stripe	evt_46_10	2025-11-26 21:12:51.919299+00	{"demo": true, "amount_cents": 5100}	2026-02-26 03:44:51.919299+00
ad11abb2-4eae-4e1d-884a-f41554f79e75	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	subscription.renewed	stripe	evt_47_1	2026-02-14 22:32:51.919299+00	{"demo": true, "amount_cents": 4750}	2026-02-26 03:44:51.919299+00
d71b12ff-7bc4-4c43-96b2-6fdd16670f94	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	subscription.cancelled	stripe	evt_47_2	2026-02-05 22:19:51.919299+00	{"demo": true, "amount_cents": 4800}	2026-02-26 03:44:51.919299+00
e8ab8df1-cc20-4ae9-935d-923657771590	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	ticket.opened	stripe	evt_47_3	2026-01-27 22:06:51.919299+00	{"demo": true, "amount_cents": 4850}	2026-02-26 03:44:51.919299+00
2f89752e-339f-4270-8225-368f98fe8825	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	ticket.resolved	stripe	evt_47_4	2026-01-18 21:53:51.919299+00	{"demo": true, "amount_cents": 4900}	2026-02-26 03:44:51.919299+00
a302e043-00df-411d-beaf-896e98f527f8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	login	stripe	evt_47_5	2026-01-09 21:40:51.919299+00	{"demo": true, "amount_cents": 4950}	2026-02-26 03:44:51.919299+00
41c12a38-1567-4173-8081-fedda85a8704	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	feature.used	stripe	evt_47_6	2025-12-31 21:27:51.919299+00	{"demo": true, "amount_cents": 5000}	2026-02-26 03:44:51.919299+00
0ec7f2b1-53ef-4852-8698-ae92b1f2b0e4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	payment.success	stripe	evt_47_7	2025-12-22 21:14:51.919299+00	{"demo": true, "amount_cents": 5050}	2026-02-26 03:44:51.919299+00
ff9a390d-47c1-44e1-be18-dff582423d8b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	payment.failed	stripe	evt_47_8	2025-12-13 21:01:51.919299+00	{"demo": true, "amount_cents": 5100}	2026-02-26 03:44:51.919299+00
01f6bc4b-28ee-48d8-82da-17096b8c0631	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	subscription.created	stripe	evt_47_9	2025-12-04 20:48:51.919299+00	{"demo": true, "amount_cents": 5150}	2026-02-26 03:44:51.919299+00
bf037532-b891-4a67-b749-11dfac370b7c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	subscription.renewed	stripe	evt_47_10	2025-11-25 20:35:51.919299+00	{"demo": true, "amount_cents": 5200}	2026-02-26 03:44:51.919299+00
2fde0554-6fc1-47b0-9d63-93486943dbe9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	subscription.cancelled	stripe	evt_48_1	2026-02-13 21:55:51.919299+00	{"demo": true, "amount_cents": 4850}	2026-02-26 03:44:51.919299+00
165a0536-cd62-4f80-b3b9-9b06acd02e2d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	ticket.opened	stripe	evt_48_2	2026-02-04 21:42:51.919299+00	{"demo": true, "amount_cents": 4900}	2026-02-26 03:44:51.919299+00
82171e52-4c63-44a9-8b32-26d02f22ee32	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	ticket.resolved	stripe	evt_48_3	2026-01-26 21:29:51.919299+00	{"demo": true, "amount_cents": 4950}	2026-02-26 03:44:51.919299+00
c8464314-5da5-425d-a2e4-0979cdf20062	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	login	stripe	evt_48_4	2026-01-17 21:16:51.919299+00	{"demo": true, "amount_cents": 5000}	2026-02-26 03:44:51.919299+00
159113f6-7e1a-4245-a0d9-d4cda43110e8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	feature.used	stripe	evt_48_5	2026-01-08 21:03:51.919299+00	{"demo": true, "amount_cents": 5050}	2026-02-26 03:44:51.919299+00
a9cd70d5-1a1c-4768-b73d-b5737f543466	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	payment.success	stripe	evt_48_6	2025-12-30 20:50:51.919299+00	{"demo": true, "amount_cents": 5100}	2026-02-26 03:44:51.919299+00
574b1e13-582c-4e9c-90b6-7caf0c131312	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	payment.failed	stripe	evt_48_7	2025-12-21 20:37:51.919299+00	{"demo": true, "amount_cents": 5150}	2026-02-26 03:44:51.919299+00
bfcf9929-7ef1-4363-ab25-92ca6f0d590f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	subscription.created	stripe	evt_48_8	2025-12-12 20:24:51.919299+00	{"demo": true, "amount_cents": 5200}	2026-02-26 03:44:51.919299+00
5eef9fd9-2460-49bc-853e-e98262193f75	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	subscription.renewed	stripe	evt_48_9	2025-12-03 20:11:51.919299+00	{"demo": true, "amount_cents": 5250}	2026-02-26 03:44:51.919299+00
a2ab8fab-059e-4aa9-bd1b-ec1ed9a54ac3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	subscription.cancelled	stripe	evt_48_10	2025-11-24 19:58:51.919299+00	{"demo": true, "amount_cents": 5300}	2026-02-26 03:44:51.919299+00
889d51a8-7738-4cba-862f-d56d9b5ac0e2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	ticket.opened	stripe	evt_49_1	2026-02-12 21:18:51.919299+00	{"demo": true, "amount_cents": 4950}	2026-02-26 03:44:51.919299+00
92e79390-79ba-4b7b-b52a-6cf6e343c8a6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	ticket.resolved	stripe	evt_49_2	2026-02-03 21:05:51.919299+00	{"demo": true, "amount_cents": 5000}	2026-02-26 03:44:51.919299+00
20171249-8a04-4c62-a67d-5ccbbadc212e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	login	stripe	evt_49_3	2026-01-25 20:52:51.919299+00	{"demo": true, "amount_cents": 5050}	2026-02-26 03:44:51.919299+00
fca29f5a-67b2-498e-8a10-3319b5b94b23	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	feature.used	stripe	evt_49_4	2026-01-16 20:39:51.919299+00	{"demo": true, "amount_cents": 5100}	2026-02-26 03:44:51.919299+00
5d20ac03-d482-4e29-bb03-d9e8da5b4a2c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	payment.success	stripe	evt_49_5	2026-01-07 20:26:51.919299+00	{"demo": true, "amount_cents": 5150}	2026-02-26 03:44:51.919299+00
0487e23a-86ad-464c-8a9f-a696644e46b7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	payment.failed	stripe	evt_49_6	2025-12-29 20:13:51.919299+00	{"demo": true, "amount_cents": 5200}	2026-02-26 03:44:51.919299+00
a122444f-67ce-4f2e-bee0-f677db7e4a4f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	subscription.created	stripe	evt_49_7	2025-12-20 20:00:51.919299+00	{"demo": true, "amount_cents": 5250}	2026-02-26 03:44:51.919299+00
5bdd7bd7-e9f4-4ec4-a663-4274670bfaa0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	subscription.renewed	stripe	evt_49_8	2025-12-11 19:47:51.919299+00	{"demo": true, "amount_cents": 5300}	2026-02-26 03:44:51.919299+00
8ddd70ff-c8e5-49eb-bd96-19db3af777dd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	subscription.cancelled	stripe	evt_49_9	2025-12-02 19:34:51.919299+00	{"demo": true, "amount_cents": 5350}	2026-02-26 03:44:51.919299+00
76883d11-9905-43d6-a01d-3ecbac0aa913	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	ticket.opened	stripe	evt_49_10	2025-11-23 19:21:51.919299+00	{"demo": true, "amount_cents": 5400}	2026-02-26 03:44:51.919299+00
9209871c-0d36-4ecb-876b-b6a152a9bbe0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	ticket.resolved	stripe	evt_50_1	2026-02-16 20:41:51.919299+00	{"demo": true, "amount_cents": 5050}	2026-02-26 03:44:51.919299+00
4529d9c4-7a91-4286-a19a-8778ad6c81e9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	login	stripe	evt_50_2	2026-02-07 20:28:51.919299+00	{"demo": true, "amount_cents": 5100}	2026-02-26 03:44:51.919299+00
085c2d63-6bf1-4228-98d2-4d43a240cde1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	feature.used	stripe	evt_50_3	2026-01-29 20:15:51.919299+00	{"demo": true, "amount_cents": 5150}	2026-02-26 03:44:51.919299+00
604fd189-a66b-4a54-bbb8-0d192ec3559c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	payment.success	stripe	evt_50_4	2026-01-20 20:02:51.919299+00	{"demo": true, "amount_cents": 5200}	2026-02-26 03:44:51.919299+00
9fa7f9ec-67bc-4479-996e-5a787e65b1ab	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	payment.failed	stripe	evt_50_5	2026-01-11 19:49:51.919299+00	{"demo": true, "amount_cents": 5250}	2026-02-26 03:44:51.919299+00
055e935f-822a-4be0-822e-9a9a12666f89	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	subscription.created	stripe	evt_50_6	2026-01-02 19:36:51.919299+00	{"demo": true, "amount_cents": 5300}	2026-02-26 03:44:51.919299+00
1258e881-8b03-408d-b8f0-5c074697e257	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	subscription.renewed	stripe	evt_50_7	2025-12-24 19:23:51.919299+00	{"demo": true, "amount_cents": 5350}	2026-02-26 03:44:51.919299+00
be84bcda-60c9-4668-804a-27b993c05bab	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	subscription.cancelled	stripe	evt_50_8	2025-12-15 19:10:51.919299+00	{"demo": true, "amount_cents": 5400}	2026-02-26 03:44:51.919299+00
4cc25c4e-19ea-42be-ba70-7a8c839eb7d8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	ticket.opened	stripe	evt_50_9	2025-12-06 18:57:51.919299+00	{"demo": true, "amount_cents": 5450}	2026-02-26 03:44:51.919299+00
384f2cae-bcc9-4b34-aecf-6b7a613ddd4e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	ticket.resolved	stripe	evt_50_10	2025-11-27 18:44:51.919299+00	{"demo": true, "amount_cents": 5500}	2026-02-26 03:44:51.919299+00
\.


--
-- Data for Name: customers; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.customers (id, org_id, external_id, source, email, name, company_name, mrr_cents, currency, first_seen_at, last_seen_at, metadata, created_at, updated_at, deleted_at) FROM stdin;
d0000000-0000-4000-a000-000000000001	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0001	stripe	customer1@example.com	Customer 1	Company B	5200	USD	2025-11-28 04:44:51.875427+00	2026-02-25 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000002	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0002	stripe	customer2@example.com	Customer 2	Company C	5400	USD	2025-11-28 05:44:51.875427+00	2026-02-24 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000003	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0003	stripe	customer3@example.com	Customer 3	Company D	5600	USD	2025-11-28 06:44:51.875427+00	2026-02-23 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000004	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0004	stripe	customer4@example.com	Customer 4	Company E	5800	USD	2025-11-28 07:44:51.875427+00	2026-02-22 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000005	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0005	stripe	customer5@example.com	Customer 5	Company F	6000	USD	2025-11-28 08:44:51.875427+00	2026-02-21 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000006	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0006	stripe	customer6@example.com	Customer 6	Company G	6200	USD	2025-11-28 09:44:51.875427+00	2026-02-20 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000007	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0007	stripe	customer7@example.com	Customer 7	Company H	6400	USD	2025-11-28 10:44:51.875427+00	2026-02-26 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000008	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0008	stripe	customer8@example.com	Customer 8	Company I	6600	USD	2025-11-28 11:44:51.875427+00	2026-02-25 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000009	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0009	stripe	customer9@example.com	Customer 9	Company J	6800	USD	2025-11-28 12:44:51.875427+00	2026-02-24 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000010	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0010	stripe	customer10@example.com	Customer 10	Company K	7000	USD	2025-11-28 13:44:51.875427+00	2026-02-23 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000011	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0011	stripe	customer11@example.com	Customer 11	Company L	7200	USD	2025-11-28 14:44:51.875427+00	2026-02-22 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000012	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0012	stripe	customer12@example.com	Customer 12	Company M	7400	USD	2025-11-28 15:44:51.875427+00	2026-02-21 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000013	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0013	stripe	customer13@example.com	Customer 13	Company N	7600	USD	2025-11-28 16:44:51.875427+00	2026-02-20 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000014	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0014	stripe	customer14@example.com	Customer 14	Company O	7800	USD	2025-11-28 17:44:51.875427+00	2026-02-26 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000015	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0015	stripe	customer15@example.com	Customer 15	Company P	8000	USD	2025-11-28 18:44:51.875427+00	2026-02-25 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000016	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0016	stripe	customer16@example.com	Customer 16	Company Q	8200	USD	2025-11-28 19:44:51.875427+00	2026-02-24 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000017	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0017	stripe	customer17@example.com	Customer 17	Company R	8400	USD	2025-11-28 20:44:51.875427+00	2026-02-23 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000018	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0018	stripe	customer18@example.com	Customer 18	Company S	8600	USD	2025-11-28 21:44:51.875427+00	2026-02-22 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000019	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0019	stripe	customer19@example.com	Customer 19	Company T	8800	USD	2025-11-28 22:44:51.875427+00	2026-02-21 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000020	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0020	stripe	customer20@example.com	Customer 20	Company U	9000	USD	2025-11-28 23:44:51.875427+00	2026-02-20 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000021	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0021	stripe	customer21@example.com	Customer 21	Company V	9200	USD	2025-11-29 00:44:51.875427+00	2026-02-26 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000022	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0022	stripe	customer22@example.com	Customer 22	Company W	9400	USD	2025-11-29 01:44:51.875427+00	2026-02-25 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000023	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0023	stripe	customer23@example.com	Customer 23	Company X	9600	USD	2025-11-29 02:44:51.875427+00	2026-02-24 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000024	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0024	stripe	customer24@example.com	Customer 24	Company Y	9800	USD	2025-11-29 03:44:51.875427+00	2026-02-23 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000025	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0025	stripe	customer25@example.com	Customer 25	Company Z	10000	USD	2025-11-29 04:44:51.875427+00	2026-02-22 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000026	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0026	stripe	customer26@example.com	Customer 26	Company A	10200	USD	2025-11-29 05:44:51.875427+00	2026-02-21 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000027	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0027	stripe	customer27@example.com	Customer 27	Company B	10400	USD	2025-11-29 06:44:51.875427+00	2026-02-20 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000028	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0028	stripe	customer28@example.com	Customer 28	Company C	10600	USD	2025-11-29 07:44:51.875427+00	2026-02-26 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000029	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0029	stripe	customer29@example.com	Customer 29	Company D	10800	USD	2025-11-29 08:44:51.875427+00	2026-02-25 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000030	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0030	stripe	customer30@example.com	Customer 30	Company E	11000	USD	2025-11-29 09:44:51.875427+00	2026-02-24 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000031	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0031	stripe	customer31@example.com	Customer 31	Company F	6100	USD	2025-11-29 10:44:51.875427+00	2026-02-23 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000032	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0032	stripe	customer32@example.com	Customer 32	Company G	6200	USD	2025-11-29 11:44:51.875427+00	2026-02-22 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000033	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0033	stripe	customer33@example.com	Customer 33	Company H	6300	USD	2025-11-29 12:44:51.875427+00	2026-02-21 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000034	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0034	stripe	customer34@example.com	Customer 34	Company I	6400	USD	2025-11-29 13:44:51.875427+00	2026-02-20 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000035	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0035	stripe	customer35@example.com	Customer 35	Company J	6500	USD	2025-11-29 14:44:51.875427+00	2026-02-26 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000036	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0036	stripe	customer36@example.com	Customer 36	Company K	6600	USD	2025-11-29 15:44:51.875427+00	2026-02-25 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000037	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0037	stripe	customer37@example.com	Customer 37	Company L	6700	USD	2025-11-29 16:44:51.875427+00	2026-02-24 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000038	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0038	stripe	customer38@example.com	Customer 38	Company M	6800	USD	2025-11-29 17:44:51.875427+00	2026-02-23 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000039	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0039	stripe	customer39@example.com	Customer 39	Company N	6900	USD	2025-11-29 18:44:51.875427+00	2026-02-22 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000040	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0040	stripe	customer40@example.com	Customer 40	Company O	7000	USD	2025-11-29 19:44:51.875427+00	2026-02-21 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000041	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0041	stripe	customer41@example.com	Customer 41	Company P	7100	USD	2025-11-29 20:44:51.875427+00	2026-02-20 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000042	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0042	stripe	customer42@example.com	Customer 42	Company Q	7200	USD	2025-11-29 21:44:51.875427+00	2026-02-26 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000043	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0043	stripe	customer43@example.com	Customer 43	Company R	3150	USD	2025-11-29 22:44:51.875427+00	2026-02-25 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000044	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0044	stripe	customer44@example.com	Customer 44	Company S	3200	USD	2025-11-29 23:44:51.875427+00	2026-02-24 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000045	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0045	stripe	customer45@example.com	Customer 45	Company T	3250	USD	2025-11-30 00:44:51.875427+00	2026-02-23 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000046	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0046	stripe	customer46@example.com	Customer 46	Company U	3300	USD	2025-11-30 01:44:51.875427+00	2026-02-22 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000047	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0047	stripe	customer47@example.com	Customer 47	Company V	3350	USD	2025-11-30 02:44:51.875427+00	2026-02-21 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000048	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0048	stripe	customer48@example.com	Customer 48	Company W	3400	USD	2025-11-30 03:44:51.875427+00	2026-02-20 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000049	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0049	stripe	customer49@example.com	Customer 49	Company X	3450	USD	2025-11-30 04:44:51.875427+00	2026-02-26 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
d0000000-0000-4000-a000-000000000050	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	cus_0050	stripe	customer50@example.com	Customer 50	Company Y	3500	USD	2025-11-30 05:44:51.875427+00	2026-02-25 03:44:51.875427+00	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	\N
\.


--
-- Data for Name: health_score_history; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.health_score_history (id, org_id, customer_id, overall_score, risk_level, factors, calculated_at, created_at) FROM stdin;
d1836101-03d1-4997-9b44-b2f1c0aeeb25	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	67	green	{"mrr_trend": 73, "payment_recency": 67, "support_tickets": 70, "usage_frequency": 66}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
2cc9b0ab-55c8-44d4-8b81-d1a1f19feff8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	69	green	{"mrr_trend": 73, "payment_recency": 69, "support_tickets": 70, "usage_frequency": 68}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
7957c4e4-774b-4dbe-9a76-4464c492e888	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	71	green	{"mrr_trend": 73, "payment_recency": 71, "support_tickets": 70, "usage_frequency": 70}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
11cc1090-af94-4d49-87cd-8bcacf60d374	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	73	green	{"mrr_trend": 73, "payment_recency": 73, "support_tickets": 70, "usage_frequency": 72}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
aa1d9638-da01-4768-b3ea-e7c4166b7a85	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	75	green	{"mrr_trend": 73, "payment_recency": 75, "support_tickets": 70, "usage_frequency": 74}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
4f706d75-64f5-4d36-87ac-ddd0837988f4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	77	green	{"mrr_trend": 73, "payment_recency": 77, "support_tickets": 70, "usage_frequency": 76}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
9fc042ea-7d70-40fe-abeb-01548e108044	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	79	green	{"mrr_trend": 73, "payment_recency": 79, "support_tickets": 70, "usage_frequency": 78}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
02edbe96-5331-4b33-a4d1-b755f598463a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	81	green	{"mrr_trend": 81, "payment_recency": 81, "support_tickets": 70, "usage_frequency": 80}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
33371b59-efd6-45a0-a783-498b09ec6739	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	83	green	{"mrr_trend": 81, "payment_recency": 83, "support_tickets": 70, "usage_frequency": 82}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
32490381-3bb4-459a-b82a-16dba75080b1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	85	green	{"mrr_trend": 81, "payment_recency": 75, "support_tickets": 70, "usage_frequency": 84}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
481b60e2-dc79-430d-a41a-c6282065227c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	76	green	{"mrr_trend": 81, "payment_recency": 77, "support_tickets": 70, "usage_frequency": 86}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
971f5486-4354-475d-9a6a-8290b8662734	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	78	green	{"mrr_trend": 81, "payment_recency": 79, "support_tickets": 70, "usage_frequency": 76}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
ba516682-bc30-414d-935e-0cacfca57554	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	80	green	{"mrr_trend": 81, "payment_recency": 81, "support_tickets": 70, "usage_frequency": 78}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
1fbac35d-7827-4d77-8fef-26d5c9d6a8a0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	82	green	{"mrr_trend": 81, "payment_recency": 83, "support_tickets": 70, "usage_frequency": 80}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
76373165-dc47-4573-b76a-33614d0fea8b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	84	green	{"mrr_trend": 81, "payment_recency": 85, "support_tickets": 85, "usage_frequency": 82}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
69b99581-af97-4d0c-a85f-4f96f7602a49	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	86	green	{"mrr_trend": 89, "payment_recency": 87, "support_tickets": 85, "usage_frequency": 84}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
27a12e06-e253-4e42-a877-f6a63c81c5ae	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	88	green	{"mrr_trend": 89, "payment_recency": 89, "support_tickets": 85, "usage_frequency": 86}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
084c5917-d1c3-4f2a-9730-05c86d55b2b7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	90	green	{"mrr_trend": 89, "payment_recency": 91, "support_tickets": 85, "usage_frequency": 88}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
456310b2-7b7c-4094-9e76-16cc22309950	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	92	green	{"mrr_trend": 89, "payment_recency": 93, "support_tickets": 85, "usage_frequency": 90}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
df49db1e-90f7-4d1d-93ed-994d87965960	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	94	green	{"mrr_trend": 89, "payment_recency": 85, "support_tickets": 85, "usage_frequency": 92}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
de9c8a23-832f-4f13-ac3e-a332f01a2328	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	96	green	{"mrr_trend": 89, "payment_recency": 87, "support_tickets": 85, "usage_frequency": 94}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
6b68fcde-ab1a-4104-92e1-becf37c3c6c7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	87	green	{"mrr_trend": 89, "payment_recency": 89, "support_tickets": 85, "usage_frequency": 96}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
f41b02c9-84a1-4346-920a-6c537a3bc6f8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	89	green	{"mrr_trend": 89, "payment_recency": 91, "support_tickets": 85, "usage_frequency": 98}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e54814d5-e2a3-42b2-aa9e-00749bb85500	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	91	green	{"mrr_trend": 97, "payment_recency": 93, "support_tickets": 85, "usage_frequency": 88}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e3a3d107-44cb-4f2f-9f53-817efd4d6a4a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	93	green	{"mrr_trend": 97, "payment_recency": 95, "support_tickets": 85, "usage_frequency": 90}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
7412a417-6e36-446b-aaa4-e7342d4ab2f8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	95	green	{"mrr_trend": 97, "payment_recency": 97, "support_tickets": 85, "usage_frequency": 92}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
8fc17e94-fdef-4144-ae31-34e52622fe19	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	97	green	{"mrr_trend": 97, "payment_recency": 99, "support_tickets": 85, "usage_frequency": 94}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
43210168-9509-4bc5-b296-8826630f5e6c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	99	green	{"mrr_trend": 97, "payment_recency": 101, "support_tickets": 85, "usage_frequency": 96}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
c9c1da99-2895-4e43-9c72-63c08a9242b9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	100	green	{"mrr_trend": 97, "payment_recency": 103, "support_tickets": 85, "usage_frequency": 98}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
4a7c8f8e-e85f-46a8-93ce-54c60055deee	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	100	green	{"mrr_trend": 97, "payment_recency": 95, "support_tickets": 100, "usage_frequency": 100}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
2c204cc3-f6bc-41fa-a3ac-512ff01069da	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	45	yellow	{"mrr_trend": 37, "payment_recency": 37, "support_tickets": 40, "usage_frequency": 42}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
95166344-29be-4707-9c91-f1ffaec5a00d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	47	yellow	{"mrr_trend": 45, "payment_recency": 39, "support_tickets": 40, "usage_frequency": 44}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
92e4353c-f2cb-42e4-8902-79ed987ffc51	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	38	yellow	{"mrr_trend": 45, "payment_recency": 41, "support_tickets": 40, "usage_frequency": 46}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
12e9363a-7ce1-42c0-9200-4386fb136209	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	40	yellow	{"mrr_trend": 45, "payment_recency": 43, "support_tickets": 40, "usage_frequency": 48}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
977c4b05-c347-40eb-8f69-c2bf7ce0e32a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	42	yellow	{"mrr_trend": 45, "payment_recency": 45, "support_tickets": 40, "usage_frequency": 50}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
1facf3d0-9c37-4f7b-9dae-d7077eaa72eb	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	44	yellow	{"mrr_trend": 45, "payment_recency": 47, "support_tickets": 40, "usage_frequency": 40}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
795215a7-4670-4e28-94a2-4464fd71d727	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	46	yellow	{"mrr_trend": 45, "payment_recency": 49, "support_tickets": 40, "usage_frequency": 42}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
3560f26d-5adb-4f7f-bf19-c7d53f70e6fd	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	48	yellow	{"mrr_trend": 45, "payment_recency": 51, "support_tickets": 40, "usage_frequency": 44}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
2ffba76f-5b74-4fa2-b3fb-4d8ee70c60c7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	50	yellow	{"mrr_trend": 45, "payment_recency": 53, "support_tickets": 40, "usage_frequency": 46}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
7c779eb5-2fc1-4c91-b622-511dd5b116d7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	52	yellow	{"mrr_trend": 53, "payment_recency": 45, "support_tickets": 40, "usage_frequency": 48}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
32cb17f1-f9cc-4ff2-8d09-c4ff2b4aab3f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	54	yellow	{"mrr_trend": 53, "payment_recency": 47, "support_tickets": 40, "usage_frequency": 50}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
c19b7105-4a57-4600-a0e6-9557674ccdea	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	56	yellow	{"mrr_trend": 53, "payment_recency": 49, "support_tickets": 40, "usage_frequency": 52}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
dc054f85-91ad-4ef3-a3dd-9fc7a737a8b0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	18	red	{"mrr_trend": 13, "payment_recency": 11, "support_tickets": 0, "usage_frequency": 14}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
536ab5b4-21f7-40aa-b662-92f408b7ce95	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	9	red	{"mrr_trend": 13, "payment_recency": 13, "support_tickets": 0, "usage_frequency": 16}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
8e651b84-2767-4c69-989f-027f3e0ae1f0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	11	red	{"mrr_trend": 13, "payment_recency": 15, "support_tickets": 15, "usage_frequency": 18}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
d11bd717-1204-4c12-85be-6993de44ca97	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	13	red	{"mrr_trend": 13, "payment_recency": 17, "support_tickets": 15, "usage_frequency": 20}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
873c609f-cff0-4d6d-bf91-5c84a369a568	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	15	red	{"mrr_trend": 13, "payment_recency": 19, "support_tickets": 15, "usage_frequency": 22}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
362ac57f-9ce8-4248-867e-df34d549717f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	17	red	{"mrr_trend": 21, "payment_recency": 21, "support_tickets": 15, "usage_frequency": 12}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
b320aea1-28d0-4965-860a-e475cac75b6a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	19	red	{"mrr_trend": 21, "payment_recency": 23, "support_tickets": 15, "usage_frequency": 14}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
9f91bd6b-efa6-43ee-9196-f33bc280d9e6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	21	red	{"mrr_trend": 21, "payment_recency": 15, "support_tickets": 15, "usage_frequency": 16}	2026-02-19 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
\.


--
-- Data for Name: health_scores; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.health_scores (id, org_id, customer_id, overall_score, risk_level, factors, calculated_at, created_at, updated_at) FROM stdin;
e0000000-0000-4000-a000-000000000001	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	71	green	{"mrr_trend": 73, "payment_recency": 67, "support_tickets": 70, "usage_frequency": 66}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000002	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	72	green	{"mrr_trend": 73, "payment_recency": 69, "support_tickets": 70, "usage_frequency": 68}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000003	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	73	green	{"mrr_trend": 73, "payment_recency": 71, "support_tickets": 70, "usage_frequency": 70}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000004	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	74	green	{"mrr_trend": 73, "payment_recency": 73, "support_tickets": 70, "usage_frequency": 72}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000005	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	75	green	{"mrr_trend": 73, "payment_recency": 75, "support_tickets": 70, "usage_frequency": 74}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000006	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	76	green	{"mrr_trend": 73, "payment_recency": 77, "support_tickets": 70, "usage_frequency": 76}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000007	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	77	green	{"mrr_trend": 73, "payment_recency": 79, "support_tickets": 70, "usage_frequency": 78}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000008	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	78	green	{"mrr_trend": 81, "payment_recency": 81, "support_tickets": 70, "usage_frequency": 80}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000009	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	79	green	{"mrr_trend": 81, "payment_recency": 83, "support_tickets": 70, "usage_frequency": 82}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000010	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	80	green	{"mrr_trend": 81, "payment_recency": 75, "support_tickets": 70, "usage_frequency": 84}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000011	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	81	green	{"mrr_trend": 81, "payment_recency": 77, "support_tickets": 70, "usage_frequency": 86}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000012	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	82	green	{"mrr_trend": 81, "payment_recency": 79, "support_tickets": 70, "usage_frequency": 76}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000013	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	83	green	{"mrr_trend": 81, "payment_recency": 81, "support_tickets": 70, "usage_frequency": 78}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000014	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	84	green	{"mrr_trend": 81, "payment_recency": 83, "support_tickets": 70, "usage_frequency": 80}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000015	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	85	green	{"mrr_trend": 81, "payment_recency": 85, "support_tickets": 85, "usage_frequency": 82}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000016	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	86	green	{"mrr_trend": 89, "payment_recency": 87, "support_tickets": 85, "usage_frequency": 84}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000017	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	87	green	{"mrr_trend": 89, "payment_recency": 89, "support_tickets": 85, "usage_frequency": 86}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000018	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	88	green	{"mrr_trend": 89, "payment_recency": 91, "support_tickets": 85, "usage_frequency": 88}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000019	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	89	green	{"mrr_trend": 89, "payment_recency": 93, "support_tickets": 85, "usage_frequency": 90}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000020	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	90	green	{"mrr_trend": 89, "payment_recency": 85, "support_tickets": 85, "usage_frequency": 92}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000021	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	91	green	{"mrr_trend": 89, "payment_recency": 87, "support_tickets": 85, "usage_frequency": 94}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000022	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	92	green	{"mrr_trend": 89, "payment_recency": 89, "support_tickets": 85, "usage_frequency": 96}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000023	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	93	green	{"mrr_trend": 89, "payment_recency": 91, "support_tickets": 85, "usage_frequency": 98}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000024	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	94	green	{"mrr_trend": 97, "payment_recency": 93, "support_tickets": 85, "usage_frequency": 88}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000025	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	95	green	{"mrr_trend": 97, "payment_recency": 95, "support_tickets": 85, "usage_frequency": 90}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000026	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	96	green	{"mrr_trend": 97, "payment_recency": 97, "support_tickets": 85, "usage_frequency": 92}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000027	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	97	green	{"mrr_trend": 97, "payment_recency": 99, "support_tickets": 85, "usage_frequency": 94}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000028	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	98	green	{"mrr_trend": 97, "payment_recency": 101, "support_tickets": 85, "usage_frequency": 96}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000029	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	99	green	{"mrr_trend": 97, "payment_recency": 103, "support_tickets": 85, "usage_frequency": 98}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000030	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	100	green	{"mrr_trend": 97, "payment_recency": 95, "support_tickets": 100, "usage_frequency": 100}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000031	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	41	yellow	{"mrr_trend": 37, "payment_recency": 37, "support_tickets": 40, "usage_frequency": 42}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000032	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	42	yellow	{"mrr_trend": 45, "payment_recency": 39, "support_tickets": 40, "usage_frequency": 44}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000033	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	43	yellow	{"mrr_trend": 45, "payment_recency": 41, "support_tickets": 40, "usage_frequency": 46}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000034	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	44	yellow	{"mrr_trend": 45, "payment_recency": 43, "support_tickets": 40, "usage_frequency": 48}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000035	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	45	yellow	{"mrr_trend": 45, "payment_recency": 45, "support_tickets": 40, "usage_frequency": 50}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000036	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	46	yellow	{"mrr_trend": 45, "payment_recency": 47, "support_tickets": 40, "usage_frequency": 40}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000037	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	47	yellow	{"mrr_trend": 45, "payment_recency": 49, "support_tickets": 40, "usage_frequency": 42}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000038	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	48	yellow	{"mrr_trend": 45, "payment_recency": 51, "support_tickets": 40, "usage_frequency": 44}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000039	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	49	yellow	{"mrr_trend": 45, "payment_recency": 53, "support_tickets": 40, "usage_frequency": 46}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000040	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	50	yellow	{"mrr_trend": 53, "payment_recency": 45, "support_tickets": 40, "usage_frequency": 48}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000041	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	51	yellow	{"mrr_trend": 53, "payment_recency": 47, "support_tickets": 40, "usage_frequency": 50}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000042	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	52	yellow	{"mrr_trend": 53, "payment_recency": 49, "support_tickets": 40, "usage_frequency": 52}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000043	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	13	red	{"mrr_trend": 13, "payment_recency": 11, "support_tickets": 0, "usage_frequency": 14}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000044	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	14	red	{"mrr_trend": 13, "payment_recency": 13, "support_tickets": 0, "usage_frequency": 16}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000045	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	15	red	{"mrr_trend": 13, "payment_recency": 15, "support_tickets": 15, "usage_frequency": 18}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000046	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	16	red	{"mrr_trend": 13, "payment_recency": 17, "support_tickets": 15, "usage_frequency": 20}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000047	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	17	red	{"mrr_trend": 13, "payment_recency": 19, "support_tickets": 15, "usage_frequency": 22}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000048	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	18	red	{"mrr_trend": 21, "payment_recency": 21, "support_tickets": 15, "usage_frequency": 12}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000049	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	19	red	{"mrr_trend": 21, "payment_recency": 23, "support_tickets": 15, "usage_frequency": 14}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e0000000-0000-4000-a000-000000000050	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	20	red	{"mrr_trend": 21, "payment_recency": 15, "support_tickets": 15, "usage_frequency": 16}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
\.


--
-- Data for Name: hubspot_companies; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.hubspot_companies (id, org_id, hubspot_company_id, name, domain, industry, number_of_employees, annual_revenue_cents, metadata, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: hubspot_contacts; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.hubspot_contacts (id, org_id, customer_id, hubspot_contact_id, email, first_name, last_name, hubspot_company_id, lifecycle_stage, lead_status, metadata, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: hubspot_deals; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.hubspot_deals (id, org_id, customer_id, hubspot_deal_id, hubspot_contact_id, deal_name, stage, amount_cents, currency, close_date, pipeline, metadata, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: integration_connections; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.integration_connections (id, org_id, provider, status, access_token_encrypted, refresh_token_encrypted, token_expires_at, external_account_id, scopes, metadata, last_sync_at, last_sync_error, created_at, updated_at) FROM stdin;
c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	stripe	active	\N	\N	\N	acct_demo_001	\N	{"connected_at": "2025-11-01T00:00:00Z"}	\N	\N	2026-02-26 03:44:51.872328+00	2026-02-26 03:44:51.872328+00
\.


--
-- Data for Name: intercom_contacts; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.intercom_contacts (id, org_id, customer_id, intercom_contact_id, email, name, role, intercom_company_id, metadata, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: intercom_conversations; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.intercom_conversations (id, org_id, customer_id, intercom_conversation_id, intercom_contact_id, state, rating, rating_remark, open, read, priority, subject, created_at_remote, updated_at_remote, closed_at, first_response_at, metadata, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: invitations; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.invitations (id, org_id, email, role, token, status, invited_by, expires_at, created_at) FROM stdin;
\.


--
-- Data for Name: notification_preferences; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.notification_preferences (id, user_id, org_id, email_enabled, in_app_enabled, digest_enabled, digest_frequency, muted_rule_ids, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: notifications; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.notifications (id, user_id, org_id, type, title, message, data, read_at, created_at) FROM stdin;
\.


--
-- Data for Name: onboarding_events; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.onboarding_events (id, org_id, step_id, event_type, occurred_at, duration_ms, metadata, created_at) FROM stdin;
\.


--
-- Data for Name: onboarding_status; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.onboarding_status (id, org_id, current_step, completed_steps, skipped_steps, step_payloads, completed_at, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: org_subscriptions; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.org_subscriptions (id, org_id, stripe_subscription_id, stripe_customer_id, plan_tier, billing_cycle, status, current_period_start, current_period_end, cancel_at_period_end, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: organizations; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.organizations (id, name, slug, plan, stripe_customer_id, created_at, updated_at, deleted_at) FROM stdin;
a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	Acme SaaS	acme-saas	pro	cus_acme_demo_001	2026-02-26 03:44:51.861634+00	2026-02-26 03:44:51.861634+00	\N
d82859c3-3111-4dea-8edb-3ba11e73473e	Debug Org	debug-org	free	\N	2026-02-26 03:47:30.14482+00	2026-02-26 03:47:30.14482+00	\N
\.


--
-- Data for Name: password_resets; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.password_resets (id, user_id, token_hash, expires_at, used_at, created_at) FROM stdin;
\.


--
-- Data for Name: refresh_tokens; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, created_at) FROM stdin;
153f0fd8-0423-43a1-b63a-9691aa3ea6d4	662bbf8d-61b2-4a70-87ee-d1bbb060e9c4	35a80ae21d31c622cc9d1b8ee1f3dfc7988f42ac7c82c8f059a868eefa5060a2	2026-03-05 03:47:30.149204+00	\N	2026-02-26 03:47:30.149561+00
f3e9b779-e248-42b0-9d93-0fc8277b60cc	662bbf8d-61b2-4a70-87ee-d1bbb060e9c4	c711af1f1039422cdec44998a7852968b24f3d6ea4368e6aea17f8ea1c34d2fc	2026-03-05 03:48:21.684556+00	\N	2026-02-26 03:48:21.685047+00
5411f3e7-0966-49bf-ad15-3fdee7807208	662bbf8d-61b2-4a70-87ee-d1bbb060e9c4	7d057ff5eaeac89eb174b4c5c833edc10494fd5572f6b7459ddf4f66d7323517	2026-03-05 03:58:43.363273+00	\N	2026-02-26 03:58:43.364017+00
\.


--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.schema_migrations (version, dirty) FROM stdin;
22	f
\.


--
-- Data for Name: scoring_configs; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.scoring_configs (id, org_id, weights, thresholds, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: stripe_payments; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.stripe_payments (id, org_id, customer_id, stripe_payment_id, amount_cents, currency, status, failure_code, failure_message, paid_at, created_at) FROM stdin;
c27161f9-9981-4d17-9412-1eeb060d15e2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	pi_demo_1_1	5200	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
67712418-cc5a-4418-8b68-138da463a326	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	pi_demo_1_2	5200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
f06023a7-bae6-458b-be0e-448bfd546fca	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	pi_demo_1_3	5200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
cdf789d3-65fc-4bb6-be2f-8d1218709be4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	pi_demo_2_1	5400	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
bfdc7fcd-8df8-4bfa-b5eb-9ec7ff26a814	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	pi_demo_2_2	5400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
47520e01-b4d8-4e8d-b300-493695a5f3b4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	pi_demo_2_3	5400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
ed327c5e-6cb3-4595-bbf5-005dce8f90be	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	pi_demo_3_1	5600	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
82c765e1-83d1-4a09-b1d8-9c7cc1dcd0b6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	pi_demo_3_2	5600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
975e483c-94b9-4af3-8397-29a99fdd008d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	pi_demo_3_3	5600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
8430ba9f-8a83-4436-83ed-0322a250b5ce	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	pi_demo_4_1	5800	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
1819f01a-86ba-43a7-86f0-7dcb4ac5ae69	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	pi_demo_4_2	5800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
95dfffbd-5a75-457d-ac65-5b2bfb0f4329	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	pi_demo_4_3	5800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
ea3ddd06-26d0-4b85-aa54-df99cc1dae99	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	pi_demo_5_1	6000	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
20889dc7-d869-442a-a7d2-8600ea99aed6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	pi_demo_5_2	6000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
2361351d-7db3-4f77-897d-146405ac3ebf	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	pi_demo_5_3	6000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
8681de3f-5007-44dd-9234-90c0967f6c61	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	pi_demo_6_1	6200	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
67b90e24-f87b-481e-9fba-908103601838	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	pi_demo_6_2	6200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
abc2fb38-bfd0-4d63-bc00-bcc43a5aa807	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	pi_demo_6_3	6200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
f73a2df6-5bc8-4947-b5c0-a2d576cb6c4b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	pi_demo_7_1	6400	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
fb443ba9-8721-417e-b44c-13a11382c3d5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	pi_demo_7_2	6400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
5163bbf4-15d5-4a79-a6fc-7e195b67d0c0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	pi_demo_7_3	6400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
bc6d8fec-8ee5-42ce-8bb0-af49b7339900	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	pi_demo_8_1	6600	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
269feca7-6c30-4f94-9bdd-ac1110eee211	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	pi_demo_8_2	6600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
81753b63-cb13-4966-9113-e040b11b2dd4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	pi_demo_8_3	6600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
49217d0d-81c7-4342-97c8-e4b4a6762eb7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	pi_demo_9_1	6800	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
4e465952-6896-4b89-9560-128601655467	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	pi_demo_9_2	6800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
3371f75f-5038-4e07-a2b4-f579a7b7effb	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	pi_demo_9_3	6800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
89375731-4a74-4226-bcb7-22f91f82a7f2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	pi_demo_10_1	7000	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
f20e8ab3-78d0-416f-a3ec-d55aa5b36198	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	pi_demo_10_2	7000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
284a1ac0-19eb-4065-bbe6-6d8ff742a765	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	pi_demo_10_3	7000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
3374b30f-9019-41b2-958e-d9df17529847	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	pi_demo_11_1	7200	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
7f419e24-1074-4b28-a06d-27cf71955305	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	pi_demo_11_2	7200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
a4791cac-3b11-4c14-92c5-fa8e3f7bdae6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	pi_demo_11_3	7200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
d2649af0-71a5-41d8-9e73-d6553d9fc029	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	pi_demo_12_1	7400	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
60cd41f7-6177-4943-87eb-b765708b9af9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	pi_demo_12_2	7400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
8421b35f-41df-43fe-bce6-83f4e21366d0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	pi_demo_12_3	7400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
67f2aa5a-9680-4e6f-9718-89be4a10e40a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	pi_demo_13_1	7600	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
80578a78-bfda-4255-8d5b-43fe9c2d5a5f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	pi_demo_13_2	7600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
425847cc-25c8-48bb-8f9e-ce02966ad51d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	pi_demo_13_3	7600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
adfcf564-2464-440c-a590-5b658fcaca6b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	pi_demo_14_1	7800	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
6c6dda5f-fd76-44df-8e4b-3ce356fd24d3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	pi_demo_14_2	7800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
c28e5cba-7ba2-4db9-8118-73fc402d2e92	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	pi_demo_14_3	7800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
728c2a50-0852-4e28-801c-9766b5b1494e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	pi_demo_15_1	8000	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
7a55b359-fedb-424f-93ac-51ec8368a4f8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	pi_demo_15_2	8000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
8c6645af-3627-4d8f-953f-30d9bc3e89ea	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	pi_demo_15_3	8000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
842d64f2-0882-494f-8762-b5d253141ba6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	pi_demo_16_1	8200	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
a3541e7e-4bbc-4b71-91f1-c5dc142b021d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	pi_demo_16_2	8200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
3dc1292e-86e5-470a-9a31-d804fe6c87f5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	pi_demo_16_3	8200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
1d15a2e1-f3ec-4860-9429-5816691a984e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	pi_demo_17_1	8400	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
85482a1c-816f-434c-83da-2a4ba22c7017	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	pi_demo_17_2	8400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
0ebe2c53-b221-418b-ae42-bc0af6f1588f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	pi_demo_17_3	8400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
0967b0c8-acf8-432e-b5a0-8b0d48293b32	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	pi_demo_18_1	8600	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
9c3476a5-f4d8-4e18-aa68-7c129d35a86b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	pi_demo_18_2	8600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
09fbe13d-bbd4-4f89-b9c1-3b14a9a53daf	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	pi_demo_18_3	8600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
40825c78-6305-4330-9fac-54a320181bb1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	pi_demo_19_1	8800	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
ef389087-e20c-4f7f-aa2b-7446eda39735	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	pi_demo_19_2	8800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
a47b158c-ab11-4c27-8c9b-b924caea5753	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	pi_demo_19_3	8800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
9febfa93-a114-4bd0-bf25-e433943b364c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	pi_demo_20_1	9000	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
f05588e2-e723-414e-bfc2-cbd35f2e5365	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	pi_demo_20_2	9000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
c700bc26-8916-49c9-bdc5-59f7c122275c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	pi_demo_20_3	9000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
9b7531ef-c8fc-4dcd-8152-1f8071656f5a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	pi_demo_21_1	9200	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
77340e54-17f6-4d96-815a-b4d2e8117a06	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	pi_demo_21_2	9200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
57b89e53-a025-4ebe-9f70-a9ab7850a69a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	pi_demo_21_3	9200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
c5c213d7-7aab-4b7b-8495-e3648bc0db30	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	pi_demo_22_1	9400	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
b6745af7-e004-4126-b0df-4fe6c7bb056f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	pi_demo_22_2	9400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
e15ee0c3-8a14-4e52-be8e-55fe01606029	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	pi_demo_22_3	9400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
4b422d74-6f14-4063-afc6-ccc80624c0a0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	pi_demo_23_1	9600	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
9f0dccd5-07dd-431a-bb26-951387373f73	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	pi_demo_23_2	9600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
cc683d0e-70d2-43ca-8fd7-f0e2ea847eae	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	pi_demo_23_3	9600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
20d19c92-0549-4b6f-8bee-1e0e9c28c345	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	pi_demo_24_1	9800	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
44558c5f-ef7b-4b77-ae9d-9846d07dafef	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	pi_demo_24_2	9800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
f930b20d-17df-4905-91a8-16ab949a361f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	pi_demo_24_3	9800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
271eb92f-eae3-4e2c-b11e-ee5bfaca078e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	pi_demo_25_1	10000	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
a0ee4cdc-dc39-4df4-82ac-a88bfcf72845	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	pi_demo_25_2	10000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
60152308-5dba-4e03-9b04-eec0e2a31eb0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	pi_demo_25_3	10000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
19d83c4e-1ac8-448f-a846-5869b5a68031	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	pi_demo_26_1	10200	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
97faca98-4084-4df3-aa10-a0aa007e0402	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	pi_demo_26_2	10200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
4fadd536-55c4-43bf-990c-02697415c80b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	pi_demo_26_3	10200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
847bd271-f1c1-4094-8799-3b8da529bdae	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	pi_demo_27_1	10400	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
80562564-0053-4ea4-8db1-81568fe7684c	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	pi_demo_27_2	10400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
3526a965-435e-4ed8-9381-1dff17540ce6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	pi_demo_27_3	10400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
4611354c-a506-477b-be12-bc083b19b0ce	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	pi_demo_28_1	10600	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
e666a22c-aac7-44c0-a866-0210b000d10f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	pi_demo_28_2	10600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
a6b6b5dc-3d3a-4934-bd87-9f795d07906e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	pi_demo_28_3	10600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
b8658f86-0e9c-4281-bcad-cecf25abd961	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	pi_demo_29_1	10800	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
49728e75-60b5-4323-94bb-c49a755eea63	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	pi_demo_29_2	10800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
89d00da8-2f1c-4821-9170-536694fe5f99	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	pi_demo_29_3	10800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
bc7ee58e-4ad5-4547-81e7-a2142504092e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	pi_demo_30_1	11000	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
c51a87a7-a3bd-4c88-85df-a5e55f5a58cb	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	pi_demo_30_2	11000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
7c3dc9eb-f419-4925-bcad-35e92452edaf	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	pi_demo_30_3	11000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
aacb6d4f-59af-4ed4-b585-727612a4f274	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	pi_demo_31_1	11200	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
270d6d16-57c1-450e-ae28-a8a8dde98cbe	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	pi_demo_31_2	11200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
fd2ebc0a-bb65-434b-8313-a286af0c5293	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	pi_demo_31_3	11200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
73b07701-c0bd-4307-bb76-c2a1eea09503	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	pi_demo_32_1	11400	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
5dc54de7-408c-4c15-b7f3-95b7e5eb262f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	pi_demo_32_2	11400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
6d28fc49-8e97-41b5-9b31-63d4674bc087	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	pi_demo_32_3	11400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
d847f964-4b59-4424-bd62-b9fd2d492bf1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	pi_demo_33_1	11600	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
62025cfd-9e78-447c-b1ab-8c61d4181eef	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	pi_demo_33_2	11600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
b988b871-3181-400e-9400-52aaace8a853	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	pi_demo_33_3	11600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
35bb578f-bebf-4de0-9c2c-cd77b0741ba5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	pi_demo_34_1	11800	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
e6e1c225-968a-4792-9c94-72253b5853fc	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	pi_demo_34_2	11800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
24a1a34d-e8fe-4ac4-8945-5920b43753d4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	pi_demo_34_3	11800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
c76dcf43-5af3-43b3-b8f5-9285fb5bbee1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	pi_demo_35_1	12000	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
089db492-65b9-42be-a3ef-f2089b9c3437	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	pi_demo_35_2	12000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
a8fc33e2-c776-4b9a-bfe2-214fe300b06d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	pi_demo_35_3	12000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
31bf9007-c86f-49ca-b1f7-78ca12388bd4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	pi_demo_36_1	12200	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
047f92d1-78ae-4123-9169-cc37aa474466	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	pi_demo_36_2	12200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
fe637f30-ab03-4bf0-a6fb-6df39346ed9d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	pi_demo_36_3	12200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
393bc87c-2590-4388-9dc3-f6ca9f135626	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	pi_demo_37_1	12400	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
a61fed3f-0965-409e-aafa-84a9d9deb9b6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	pi_demo_37_2	12400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
70035e87-9d1b-44ce-b552-252d79d9fa12	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	pi_demo_37_3	12400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
ec627fe0-ce04-416d-baa7-90ddccda6e2f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	pi_demo_38_1	12600	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
42e14998-537b-48e2-9e91-99f437843325	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	pi_demo_38_2	12600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
fde37bbc-a033-4b7b-8827-1e72e97e53d1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	pi_demo_38_3	12600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
6b094e44-7b44-457d-8e01-08e88aeabea7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	pi_demo_39_1	12800	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
c19435d5-f2e4-4a39-b285-8589062e4bb1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	pi_demo_39_2	12800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
8ef122d0-b918-4c91-9a32-049611c8d174	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	pi_demo_39_3	12800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
e9dbbf14-8d49-4d67-aa56-902963a38f8d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	pi_demo_40_1	13000	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
7fceed31-ffec-423e-8ee7-14fd6e296400	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	pi_demo_40_2	13000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
ada76922-83ac-4236-bd41-627ea78f0d23	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	pi_demo_40_3	13000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
91bfd6ed-eb46-4732-a027-3a7d23f63a4b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	pi_demo_41_1	13200	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
3958b0cc-4cd1-48c6-b66f-717551768b52	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	pi_demo_41_2	13200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
8f165551-7010-457b-b0c1-e8fe10246a36	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	pi_demo_41_3	13200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
a4e7a979-b16c-4fc4-b100-61adf49349b2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	pi_demo_42_1	13400	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
661ec3d4-69c5-4fd5-b772-341c04f60b88	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	pi_demo_42_2	13400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
6b074b0f-ea2f-4675-aeb9-e81f6663aab2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	pi_demo_42_3	13400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
ba154ad1-e514-46f8-a204-168ff826a261	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	pi_demo_43_1	13600	USD	succeeded	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
87eeed92-a90a-4483-bb5e-9dafa04fd0d0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	pi_demo_43_2	13600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
40882c77-8a44-489a-a2da-d23831a442a7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	pi_demo_43_3	13600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
db71f3f4-21e0-4fc6-90c5-7e0f0dacaaca	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	pi_demo_44_1	13800	USD	failed	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
cf92d9de-f02d-4a99-b077-5c017cd2add4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	pi_demo_44_2	13800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
70cc3939-d1a2-4711-9457-41831f2bce0b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	pi_demo_44_3	13800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
f0afc7fc-860c-4a5c-a3ad-0db9f0fa8af1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	pi_demo_45_1	14000	USD	failed	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
d401564d-4cdc-4fd6-8425-0517f0fa0c53	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	pi_demo_45_2	14000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
4a03c783-c83c-468d-992c-b0f8e8651e9a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	pi_demo_45_3	14000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
146b9ef2-0192-4a6e-85b2-8bc5eff32129	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	pi_demo_46_1	14200	USD	failed	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
40781fd6-1c5b-4809-8d3f-19af83e6863b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	pi_demo_46_2	14200	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
8f24fdc7-2745-438c-9db3-60e79da2bc99	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	pi_demo_46_3	14200	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
277843eb-701f-48c5-b70a-0c51f4583f79	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	pi_demo_47_1	14400	USD	failed	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
f004de80-87ea-42d7-9180-f04d95f21130	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	pi_demo_47_2	14400	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
b21c378b-dfcd-4c85-90d0-880ec4d807f2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	pi_demo_47_3	14400	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
db36ae3a-6e4a-42d7-b01e-297d6bee6912	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	pi_demo_48_1	14600	USD	failed	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
bcbacfcb-e6ce-4e40-a80b-a4d5e84a959b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	pi_demo_48_2	14600	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
ef69f322-d13a-4342-9323-cd31b3dd3d21	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	pi_demo_48_3	14600	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
f87818e5-dc86-4d20-8432-1f7f7d40c2b2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	pi_demo_49_1	14800	USD	failed	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
7f532b11-cb8f-43ad-8df2-5a6d96a1c1c8	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	pi_demo_49_2	14800	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
cb301e36-6069-4811-8ae1-5aeeb57c1207	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	pi_demo_49_3	14800	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
47dd8670-2d32-400e-9787-590f89e99c0f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	pi_demo_50_1	15000	USD	failed	\N	\N	2026-01-27 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
058488f5-61e4-43b7-8742-1ff62f8e49ac	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	pi_demo_50_2	15000	USD	succeeded	\N	\N	2025-12-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
36bf6b7c-8d99-4ce0-9cd6-c643abeb017e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	pi_demo_50_3	15000	USD	succeeded	\N	\N	2025-11-28 03:44:51.98337+00	2026-02-26 03:44:51.98337+00
\.


--
-- Data for Name: stripe_subscriptions; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.stripe_subscriptions (id, org_id, customer_id, stripe_subscription_id, status, plan_name, amount_cents, currency, "interval", current_period_start, current_period_end, canceled_at, metadata, created_at, updated_at) FROM stdin;
4d5b8324-4685-491c-b414-fc590a4e2e48	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000001	sub_demo_0001	active	Pro	5200	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
d740baa9-237f-49e8-8916-3af79d00f0c7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000002	sub_demo_0002	active	Pro	5400	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
eb6eca0d-c792-4da4-9e82-d92b82e2dbec	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000003	sub_demo_0003	active	Pro	5600	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
175087b6-0dda-4482-9a9f-44a7ae536422	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000004	sub_demo_0004	active	Pro	5800	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
bfc98ef3-09f7-40ba-b181-d832ed5a5015	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000005	sub_demo_0005	active	Pro	6000	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
6c60be33-7b88-4526-bdeb-86223481c738	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000006	sub_demo_0006	active	Pro	6200	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
7d022c2e-d60e-4251-9f2f-1003d0f2a449	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000007	sub_demo_0007	active	Pro	6400	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
3cc247f9-7096-4cf0-b537-885e686d7c2f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000008	sub_demo_0008	active	Pro	6600	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
eedfda42-dca7-4848-b025-c6430c5431ee	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000009	sub_demo_0009	active	Pro	6800	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
33f0c975-1de2-4760-974f-0312c48fa7e0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000010	sub_demo_0010	active	Pro	7000	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
480f00d1-921a-4819-a3ae-8dc2d7beb7cb	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000011	sub_demo_0011	active	Pro	7200	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
4e09cbeb-26b1-48fe-954f-8b452c8f9ba2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000012	sub_demo_0012	active	Pro	7400	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e782b42b-658d-439d-bc47-4a3ff35e6ce3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000013	sub_demo_0013	active	Pro	7600	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
a97099d1-1edc-4f56-9e05-aaf51c6eaaf1	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000014	sub_demo_0014	active	Pro	7800	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
ddad333a-af47-45a7-90eb-3d495752ac5e	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000015	sub_demo_0015	active	Pro	8000	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
d762c1da-4114-41a3-9425-be7ae0f3861f	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000016	sub_demo_0016	active	Enterprise	8200	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
374d538d-2e81-419f-ba7d-a03a28c031d7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000017	sub_demo_0017	active	Enterprise	8400	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
8ec48d3c-f42f-45fa-885f-e34b8077ddb9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000018	sub_demo_0018	active	Enterprise	8600	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
7e23013c-5255-43fc-ab87-131bda268b2a	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000019	sub_demo_0019	active	Enterprise	8800	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
c3d38bf2-075e-4b86-9d7d-767f9909fce7	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000020	sub_demo_0020	active	Enterprise	9000	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
cd120b50-c800-4272-aaec-f14656a372a0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000021	sub_demo_0021	active	Enterprise	9200	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
6f61cc96-ef9d-4862-890a-af53773ed78d	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000022	sub_demo_0022	active	Enterprise	9400	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
7e7b46f2-860b-4b82-90c4-6a57bbea6fe4	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000023	sub_demo_0023	active	Enterprise	9600	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
5382dfa5-654c-4a94-acaf-5471894f1a6b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000024	sub_demo_0024	active	Enterprise	9800	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
a43b00f0-2084-42c9-8600-e29c4f216651	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000025	sub_demo_0025	active	Enterprise	10000	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
77f38d16-be47-4a2f-8946-e726d9b339b9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000026	sub_demo_0026	active	Enterprise	10200	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
37afba9f-55d1-4c14-9f0f-d6283bf95371	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000027	sub_demo_0027	active	Enterprise	10400	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
a5e0c4c3-a0ed-40cf-b24e-ae2eb2046967	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000028	sub_demo_0028	active	Enterprise	10600	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
9356e677-4716-43e6-ba9f-9e4550af8945	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000029	sub_demo_0029	active	Enterprise	10800	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
a7928f3c-6803-4912-84e4-e8ace8a932e5	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000030	sub_demo_0030	active	Enterprise	11000	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
45ee414e-27cc-4b73-8ca4-7f96ac3612d2	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000031	sub_demo_0031	active	Pro	6100	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
fbc1ca45-1272-4f2a-b0a7-c65a756a0377	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000032	sub_demo_0032	active	Pro	6200	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e493b6b5-604f-4b20-9cf6-6951440de883	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000033	sub_demo_0033	active	Pro	6300	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
a7d178b9-08fc-45ec-ae23-fe958365ab28	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000034	sub_demo_0034	active	Pro	6400	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
5b717b43-09e2-40e6-89b3-7e8cc047dba0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000035	sub_demo_0035	active	Pro	6500	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
49e60e27-8f7e-400b-9a11-3d8140117299	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000036	sub_demo_0036	active	Pro	6600	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
ba04a3e8-7063-4056-b7af-685b88a836e9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000037	sub_demo_0037	active	Pro	6700	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
647cb9b8-4631-47e5-9240-c152567666ba	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000038	sub_demo_0038	active	Pro	6800	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
8f8d7b55-8943-4f09-8b4e-ff20547415ea	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000039	sub_demo_0039	active	Pro	6900	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e84d3009-727f-45cc-816e-dbd22e1885b0	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000040	sub_demo_0040	active	Pro	7000	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
a450b896-a910-4e2d-a95b-db0b3d6a15ff	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000041	sub_demo_0041	active	Pro	7100	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
cc177b8c-848a-4fb4-9799-d65cac864907	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000042	sub_demo_0042	active	Pro	7200	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
5ce1e9ad-fd6b-4fec-a39f-da03feaa3596	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000043	sub_demo_0043	canceled	Starter	3150	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e2a6f11a-67a2-40e3-a4e4-8b6d7cbc15db	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000044	sub_demo_0044	canceled	Starter	3200	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
23072de0-3b9d-4c33-ba80-073efecccf37	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000045	sub_demo_0045	canceled	Starter	3250	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
0ecfe1dc-db64-470b-a86b-61c433f40cd3	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000046	sub_demo_0046	canceled	Starter	3300	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
30e09393-6782-48ec-9aef-7bb1633e70e6	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000047	sub_demo_0047	canceled	Starter	3350	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
e1b3537d-c3a1-4234-99f6-e1b821cb87a9	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000048	sub_demo_0048	canceled	Starter	3400	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
bc9425de-0ec4-4236-a5ed-60fa89e8d34b	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000049	sub_demo_0049	canceled	Starter	3450	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
c80013af-a69a-49cf-b3c6-d8ad7255b452	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	d0000000-0000-4000-a000-000000000050	sub_demo_0050	canceled	Starter	3500	USD	month	2026-02-01 00:00:00+00	2026-03-01 00:00:00+00	\N	{}	2026-02-26 03:44:51.875427+00	2026-02-26 03:44:51.875427+00
\.


--
-- Data for Name: user_organizations; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.user_organizations (user_id, org_id, role, created_at) FROM stdin;
b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	owner	2026-02-26 03:44:51.868047+00
b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a02	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	admin	2026-02-26 03:44:51.868047+00
b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a03	a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11	member	2026-02-26 03:44:51.868047+00
662bbf8d-61b2-4a70-87ee-d1bbb060e9c4	d82859c3-3111-4dea-8edb-3ba11e73473e	owner	2026-02-26 03:47:30.14482+00
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: pulsescore
--

COPY public.users (id, email, password_hash, first_name, last_name, avatar_url, email_verified, created_at, updated_at, deleted_at) FROM stdin;
b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a01	owner@acme.com	$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy	Alice	Owner	about:blank	t	2026-02-26 03:44:51.865742+00	2026-02-26 03:46:33.5857+00	\N
b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a02	admin@acme.com	$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy	Bob	Admin	about:blank	t	2026-02-26 03:44:51.865742+00	2026-02-26 03:46:33.5857+00	\N
b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a03	member@acme.com	$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy	Carol	Member	about:blank	t	2026-02-26 03:44:51.865742+00	2026-02-26 03:46:33.5857+00	\N
662bbf8d-61b2-4a70-87ee-d1bbb060e9c4	debug-login-fix@example.com	$2a$12$3O80McBsisYrfspPqDLPwOe5AyGbelt6B.DAXFQokrgSJu2/fWkEy	\N	\N	\N	f	2026-02-26 03:47:30.14482+00	2026-02-26 03:58:43.358599+00	\N
\.


--
-- Name: alert_history alert_history_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.alert_history
    ADD CONSTRAINT alert_history_pkey PRIMARY KEY (id);


--
-- Name: alert_rules alert_rules_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.alert_rules
    ADD CONSTRAINT alert_rules_pkey PRIMARY KEY (id);


--
-- Name: billing_webhook_events billing_webhook_events_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.billing_webhook_events
    ADD CONSTRAINT billing_webhook_events_pkey PRIMARY KEY (event_id);


--
-- Name: customer_events customer_events_org_id_source_external_event_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.customer_events
    ADD CONSTRAINT customer_events_org_id_source_external_event_id_key UNIQUE (org_id, source, external_event_id);


--
-- Name: customer_events customer_events_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.customer_events
    ADD CONSTRAINT customer_events_pkey PRIMARY KEY (id);


--
-- Name: customers customers_org_id_source_external_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.customers
    ADD CONSTRAINT customers_org_id_source_external_id_key UNIQUE (org_id, source, external_id);


--
-- Name: customers customers_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.customers
    ADD CONSTRAINT customers_pkey PRIMARY KEY (id);


--
-- Name: health_score_history health_score_history_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.health_score_history
    ADD CONSTRAINT health_score_history_pkey PRIMARY KEY (id);


--
-- Name: health_scores health_scores_customer_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.health_scores
    ADD CONSTRAINT health_scores_customer_id_key UNIQUE (customer_id);


--
-- Name: health_scores health_scores_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.health_scores
    ADD CONSTRAINT health_scores_pkey PRIMARY KEY (id);


--
-- Name: hubspot_companies hubspot_companies_org_id_hubspot_company_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_companies
    ADD CONSTRAINT hubspot_companies_org_id_hubspot_company_id_key UNIQUE (org_id, hubspot_company_id);


--
-- Name: hubspot_companies hubspot_companies_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_companies
    ADD CONSTRAINT hubspot_companies_pkey PRIMARY KEY (id);


--
-- Name: hubspot_contacts hubspot_contacts_org_id_hubspot_contact_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_contacts
    ADD CONSTRAINT hubspot_contacts_org_id_hubspot_contact_id_key UNIQUE (org_id, hubspot_contact_id);


--
-- Name: hubspot_contacts hubspot_contacts_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_contacts
    ADD CONSTRAINT hubspot_contacts_pkey PRIMARY KEY (id);


--
-- Name: hubspot_deals hubspot_deals_org_id_hubspot_deal_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_deals
    ADD CONSTRAINT hubspot_deals_org_id_hubspot_deal_id_key UNIQUE (org_id, hubspot_deal_id);


--
-- Name: hubspot_deals hubspot_deals_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_deals
    ADD CONSTRAINT hubspot_deals_pkey PRIMARY KEY (id);


--
-- Name: integration_connections integration_connections_org_id_provider_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.integration_connections
    ADD CONSTRAINT integration_connections_org_id_provider_key UNIQUE (org_id, provider);


--
-- Name: integration_connections integration_connections_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.integration_connections
    ADD CONSTRAINT integration_connections_pkey PRIMARY KEY (id);


--
-- Name: intercom_contacts intercom_contacts_org_id_intercom_contact_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.intercom_contacts
    ADD CONSTRAINT intercom_contacts_org_id_intercom_contact_id_key UNIQUE (org_id, intercom_contact_id);


--
-- Name: intercom_contacts intercom_contacts_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.intercom_contacts
    ADD CONSTRAINT intercom_contacts_pkey PRIMARY KEY (id);


--
-- Name: intercom_conversations intercom_conversations_org_id_intercom_conversation_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.intercom_conversations
    ADD CONSTRAINT intercom_conversations_org_id_intercom_conversation_id_key UNIQUE (org_id, intercom_conversation_id);


--
-- Name: intercom_conversations intercom_conversations_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.intercom_conversations
    ADD CONSTRAINT intercom_conversations_pkey PRIMARY KEY (id);


--
-- Name: invitations invitations_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_pkey PRIMARY KEY (id);


--
-- Name: invitations invitations_token_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_token_key UNIQUE (token);


--
-- Name: notification_preferences notification_preferences_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.notification_preferences
    ADD CONSTRAINT notification_preferences_pkey PRIMARY KEY (id);


--
-- Name: notification_preferences notification_preferences_user_id_org_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.notification_preferences
    ADD CONSTRAINT notification_preferences_user_id_org_id_key UNIQUE (user_id, org_id);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: onboarding_events onboarding_events_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.onboarding_events
    ADD CONSTRAINT onboarding_events_pkey PRIMARY KEY (id);


--
-- Name: onboarding_status onboarding_status_org_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.onboarding_status
    ADD CONSTRAINT onboarding_status_org_id_key UNIQUE (org_id);


--
-- Name: onboarding_status onboarding_status_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.onboarding_status
    ADD CONSTRAINT onboarding_status_pkey PRIMARY KEY (id);


--
-- Name: org_subscriptions org_subscriptions_org_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.org_subscriptions
    ADD CONSTRAINT org_subscriptions_org_id_key UNIQUE (org_id);


--
-- Name: org_subscriptions org_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.org_subscriptions
    ADD CONSTRAINT org_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: org_subscriptions org_subscriptions_stripe_subscription_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.org_subscriptions
    ADD CONSTRAINT org_subscriptions_stripe_subscription_id_key UNIQUE (stripe_subscription_id);


--
-- Name: organizations organizations_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);


--
-- Name: organizations organizations_slug_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_slug_key UNIQUE (slug);


--
-- Name: password_resets password_resets_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.password_resets
    ADD CONSTRAINT password_resets_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);


--
-- Name: refresh_tokens refresh_tokens_token_hash_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_token_hash_key UNIQUE (token_hash);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: scoring_configs scoring_configs_org_unique; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.scoring_configs
    ADD CONSTRAINT scoring_configs_org_unique UNIQUE (org_id);


--
-- Name: scoring_configs scoring_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.scoring_configs
    ADD CONSTRAINT scoring_configs_pkey PRIMARY KEY (id);


--
-- Name: stripe_payments stripe_payments_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.stripe_payments
    ADD CONSTRAINT stripe_payments_pkey PRIMARY KEY (id);


--
-- Name: stripe_payments stripe_payments_stripe_payment_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.stripe_payments
    ADD CONSTRAINT stripe_payments_stripe_payment_id_key UNIQUE (stripe_payment_id);


--
-- Name: stripe_subscriptions stripe_subscriptions_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.stripe_subscriptions
    ADD CONSTRAINT stripe_subscriptions_pkey PRIMARY KEY (id);


--
-- Name: stripe_subscriptions stripe_subscriptions_stripe_subscription_id_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.stripe_subscriptions
    ADD CONSTRAINT stripe_subscriptions_stripe_subscription_id_key UNIQUE (stripe_subscription_id);


--
-- Name: user_organizations user_organizations_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.user_organizations
    ADD CONSTRAINT user_organizations_pkey PRIMARY KEY (user_id, org_id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_alert_history_org_sent; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_alert_history_org_sent ON public.alert_history USING btree (org_id, sent_at DESC);


--
-- Name: idx_alert_history_rule; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_alert_history_rule ON public.alert_history USING btree (alert_rule_id);


--
-- Name: idx_alert_history_sendgrid_msg; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_alert_history_sendgrid_msg ON public.alert_history USING btree (sendgrid_message_id) WHERE (sendgrid_message_id IS NOT NULL);


--
-- Name: idx_alert_rules_org_active; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_alert_rules_org_active ON public.alert_rules USING btree (org_id, is_active);


--
-- Name: idx_billing_webhook_events_processed_at; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_billing_webhook_events_processed_at ON public.billing_webhook_events USING btree (processed_at DESC);


--
-- Name: idx_customer_events_customer_time; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_customer_events_customer_time ON public.customer_events USING btree (customer_id, occurred_at DESC);


--
-- Name: idx_customer_events_org_time; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_customer_events_org_time ON public.customer_events USING btree (org_id, occurred_at DESC);


--
-- Name: idx_customer_events_org_type; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_customer_events_org_type ON public.customer_events USING btree (org_id, event_type);


--
-- Name: idx_customers_org_email; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_customers_org_email ON public.customers USING btree (org_id, email);


--
-- Name: idx_customers_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_customers_org_id ON public.customers USING btree (org_id);


--
-- Name: idx_customers_org_mrr; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_customers_org_mrr ON public.customers USING btree (org_id, mrr_cents);


--
-- Name: idx_customers_org_source; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_customers_org_source ON public.customers USING btree (org_id, source);


--
-- Name: idx_health_score_history_customer_time; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_health_score_history_customer_time ON public.health_score_history USING btree (customer_id, calculated_at DESC);


--
-- Name: idx_health_score_history_org_time; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_health_score_history_org_time ON public.health_score_history USING btree (org_id, calculated_at DESC);


--
-- Name: idx_health_scores_org_risk; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_health_scores_org_risk ON public.health_scores USING btree (org_id, risk_level);


--
-- Name: idx_hubspot_companies_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_hubspot_companies_org_id ON public.hubspot_companies USING btree (org_id);


--
-- Name: idx_hubspot_contacts_customer_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_hubspot_contacts_customer_id ON public.hubspot_contacts USING btree (customer_id);


--
-- Name: idx_hubspot_contacts_email; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_hubspot_contacts_email ON public.hubspot_contacts USING btree (email);


--
-- Name: idx_hubspot_contacts_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_hubspot_contacts_org_id ON public.hubspot_contacts USING btree (org_id);


--
-- Name: idx_hubspot_deals_customer_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_hubspot_deals_customer_id ON public.hubspot_deals USING btree (customer_id);


--
-- Name: idx_hubspot_deals_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_hubspot_deals_org_id ON public.hubspot_deals USING btree (org_id);


--
-- Name: idx_integration_connections_provider_status; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_integration_connections_provider_status ON public.integration_connections USING btree (provider, status);


--
-- Name: idx_intercom_contacts_customer_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_intercom_contacts_customer_id ON public.intercom_contacts USING btree (customer_id);


--
-- Name: idx_intercom_contacts_email; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_intercom_contacts_email ON public.intercom_contacts USING btree (email);


--
-- Name: idx_intercom_contacts_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_intercom_contacts_org_id ON public.intercom_contacts USING btree (org_id);


--
-- Name: idx_intercom_conversations_created_remote; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_intercom_conversations_created_remote ON public.intercom_conversations USING btree (org_id, created_at_remote);


--
-- Name: idx_intercom_conversations_customer_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_intercom_conversations_customer_id ON public.intercom_conversations USING btree (customer_id);


--
-- Name: idx_intercom_conversations_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_intercom_conversations_org_id ON public.intercom_conversations USING btree (org_id);


--
-- Name: idx_intercom_conversations_state; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_intercom_conversations_state ON public.intercom_conversations USING btree (org_id, state);


--
-- Name: idx_invitations_email; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_invitations_email ON public.invitations USING btree (email);


--
-- Name: idx_invitations_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_invitations_org_id ON public.invitations USING btree (org_id);


--
-- Name: idx_invitations_token; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_invitations_token ON public.invitations USING btree (token) WHERE ((status)::text = 'pending'::text);


--
-- Name: idx_notification_prefs_user_org; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_notification_prefs_user_org ON public.notification_preferences USING btree (user_id, org_id);


--
-- Name: idx_notifications_unread; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_notifications_unread ON public.notifications USING btree (user_id, org_id) WHERE (read_at IS NULL);


--
-- Name: idx_notifications_user_org; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_notifications_user_org ON public.notifications USING btree (user_id, org_id, created_at DESC);


--
-- Name: idx_onboarding_events_org_step_type; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_onboarding_events_org_step_type ON public.onboarding_events USING btree (org_id, step_id, event_type);


--
-- Name: idx_onboarding_events_org_time; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_onboarding_events_org_time ON public.onboarding_events USING btree (org_id, occurred_at DESC);


--
-- Name: idx_onboarding_status_completed_at; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_onboarding_status_completed_at ON public.onboarding_status USING btree (completed_at);


--
-- Name: idx_onboarding_status_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_onboarding_status_org_id ON public.onboarding_status USING btree (org_id);


--
-- Name: idx_org_subscriptions_customer; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_org_subscriptions_customer ON public.org_subscriptions USING btree (stripe_customer_id) WHERE (stripe_customer_id IS NOT NULL);


--
-- Name: idx_org_subscriptions_org_status; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_org_subscriptions_org_status ON public.org_subscriptions USING btree (org_id, status);


--
-- Name: idx_organizations_stripe_customer_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_organizations_stripe_customer_id ON public.organizations USING btree (stripe_customer_id) WHERE (stripe_customer_id IS NOT NULL);


--
-- Name: idx_password_resets_token_hash; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_password_resets_token_hash ON public.password_resets USING btree (token_hash);


--
-- Name: idx_password_resets_user_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_password_resets_user_id ON public.password_resets USING btree (user_id);


--
-- Name: idx_refresh_tokens_token_hash; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_refresh_tokens_token_hash ON public.refresh_tokens USING btree (token_hash) WHERE (revoked_at IS NULL);


--
-- Name: idx_refresh_tokens_user_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_refresh_tokens_user_id ON public.refresh_tokens USING btree (user_id);


--
-- Name: idx_scoring_configs_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_scoring_configs_org_id ON public.scoring_configs USING btree (org_id);


--
-- Name: idx_stripe_payments_customer_status; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_stripe_payments_customer_status ON public.stripe_payments USING btree (customer_id, status);


--
-- Name: idx_stripe_payments_org_status; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_stripe_payments_org_status ON public.stripe_payments USING btree (org_id, status);


--
-- Name: idx_stripe_payments_paid_at; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_stripe_payments_paid_at ON public.stripe_payments USING btree (paid_at DESC);


--
-- Name: idx_stripe_subscriptions_customer_status; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_stripe_subscriptions_customer_status ON public.stripe_subscriptions USING btree (customer_id, status);


--
-- Name: idx_stripe_subscriptions_org_status; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_stripe_subscriptions_org_status ON public.stripe_subscriptions USING btree (org_id, status);


--
-- Name: idx_user_organizations_org_id; Type: INDEX; Schema: public; Owner: pulsescore
--

CREATE INDEX idx_user_organizations_org_id ON public.user_organizations USING btree (org_id);


--
-- Name: alert_rules set_alert_rules_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_alert_rules_updated_at BEFORE UPDATE ON public.alert_rules FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: customers set_customers_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_customers_updated_at BEFORE UPDATE ON public.customers FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: health_scores set_health_scores_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_health_scores_updated_at BEFORE UPDATE ON public.health_scores FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: integration_connections set_integration_connections_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_integration_connections_updated_at BEFORE UPDATE ON public.integration_connections FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: notification_preferences set_notification_preferences_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_notification_preferences_updated_at BEFORE UPDATE ON public.notification_preferences FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: onboarding_status set_onboarding_status_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_onboarding_status_updated_at BEFORE UPDATE ON public.onboarding_status FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: org_subscriptions set_org_subscriptions_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_org_subscriptions_updated_at BEFORE UPDATE ON public.org_subscriptions FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: organizations set_organizations_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_organizations_updated_at BEFORE UPDATE ON public.organizations FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: stripe_subscriptions set_stripe_subscriptions_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_stripe_subscriptions_updated_at BEFORE UPDATE ON public.stripe_subscriptions FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: users set_users_updated_at; Type: TRIGGER; Schema: public; Owner: pulsescore
--

CREATE TRIGGER set_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.trigger_set_updated_at();


--
-- Name: alert_history alert_history_alert_rule_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.alert_history
    ADD CONSTRAINT alert_history_alert_rule_id_fkey FOREIGN KEY (alert_rule_id) REFERENCES public.alert_rules(id) ON DELETE CASCADE;


--
-- Name: alert_history alert_history_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.alert_history
    ADD CONSTRAINT alert_history_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE SET NULL;


--
-- Name: alert_history alert_history_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.alert_history
    ADD CONSTRAINT alert_history_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: alert_rules alert_rules_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.alert_rules
    ADD CONSTRAINT alert_rules_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: alert_rules alert_rules_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.alert_rules
    ADD CONSTRAINT alert_rules_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: customer_events customer_events_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.customer_events
    ADD CONSTRAINT customer_events_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE;


--
-- Name: customer_events customer_events_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.customer_events
    ADD CONSTRAINT customer_events_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: customers customers_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.customers
    ADD CONSTRAINT customers_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: health_score_history health_score_history_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.health_score_history
    ADD CONSTRAINT health_score_history_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE;


--
-- Name: health_score_history health_score_history_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.health_score_history
    ADD CONSTRAINT health_score_history_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: health_scores health_scores_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.health_scores
    ADD CONSTRAINT health_scores_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE;


--
-- Name: health_scores health_scores_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.health_scores
    ADD CONSTRAINT health_scores_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: hubspot_companies hubspot_companies_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_companies
    ADD CONSTRAINT hubspot_companies_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: hubspot_contacts hubspot_contacts_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_contacts
    ADD CONSTRAINT hubspot_contacts_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE SET NULL;


--
-- Name: hubspot_contacts hubspot_contacts_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_contacts
    ADD CONSTRAINT hubspot_contacts_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: hubspot_deals hubspot_deals_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_deals
    ADD CONSTRAINT hubspot_deals_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE SET NULL;


--
-- Name: hubspot_deals hubspot_deals_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.hubspot_deals
    ADD CONSTRAINT hubspot_deals_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: integration_connections integration_connections_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.integration_connections
    ADD CONSTRAINT integration_connections_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: intercom_contacts intercom_contacts_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.intercom_contacts
    ADD CONSTRAINT intercom_contacts_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE SET NULL;


--
-- Name: intercom_contacts intercom_contacts_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.intercom_contacts
    ADD CONSTRAINT intercom_contacts_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: intercom_conversations intercom_conversations_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.intercom_conversations
    ADD CONSTRAINT intercom_conversations_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE SET NULL;


--
-- Name: intercom_conversations intercom_conversations_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.intercom_conversations
    ADD CONSTRAINT intercom_conversations_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: invitations invitations_invited_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_invited_by_fkey FOREIGN KEY (invited_by) REFERENCES public.users(id);


--
-- Name: invitations invitations_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.invitations
    ADD CONSTRAINT invitations_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: notification_preferences notification_preferences_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.notification_preferences
    ADD CONSTRAINT notification_preferences_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: notification_preferences notification_preferences_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.notification_preferences
    ADD CONSTRAINT notification_preferences_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: notifications notifications_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: notifications notifications_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: onboarding_events onboarding_events_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.onboarding_events
    ADD CONSTRAINT onboarding_events_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: onboarding_status onboarding_status_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.onboarding_status
    ADD CONSTRAINT onboarding_status_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: org_subscriptions org_subscriptions_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.org_subscriptions
    ADD CONSTRAINT org_subscriptions_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: password_resets password_resets_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.password_resets
    ADD CONSTRAINT password_resets_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: refresh_tokens refresh_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.refresh_tokens
    ADD CONSTRAINT refresh_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: scoring_configs scoring_configs_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.scoring_configs
    ADD CONSTRAINT scoring_configs_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: stripe_payments stripe_payments_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.stripe_payments
    ADD CONSTRAINT stripe_payments_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE;


--
-- Name: stripe_payments stripe_payments_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.stripe_payments
    ADD CONSTRAINT stripe_payments_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: stripe_subscriptions stripe_subscriptions_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.stripe_subscriptions
    ADD CONSTRAINT stripe_subscriptions_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customers(id) ON DELETE CASCADE;


--
-- Name: stripe_subscriptions stripe_subscriptions_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.stripe_subscriptions
    ADD CONSTRAINT stripe_subscriptions_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: user_organizations user_organizations_org_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.user_organizations
    ADD CONSTRAINT user_organizations_org_id_fkey FOREIGN KEY (org_id) REFERENCES public.organizations(id) ON DELETE CASCADE;


--
-- Name: user_organizations user_organizations_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: pulsescore
--

ALTER TABLE ONLY public.user_organizations
    ADD CONSTRAINT user_organizations_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

\unrestrict NKAhvuYSMdrGNf40aXMBAMdnUAhlcRc0KCV0BdoTXbTMhJat3iUUggWmawaMKAv

