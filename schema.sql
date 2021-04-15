-- event store
create table if not exists events (
	aggregate_id uuid not null,
	created_at timestamp with time zone not null,
	correlation_id uuid not null,
	event_id varchar(64) not null,
	payload jsonb not null
);

-- most common query should benefit from this
create index events_agg_time_idx ON events (aggregate_id, created_at);

-- users view only schema, used by API, populated by denormalizer, could be different DB completely
create table if not exists users (
	id uuid,
	email varchar(128) not null,
    password varchar(128) not null,
    enabled bool not null,
    last_event_time timestamp not null,
    last_correlation_id uuid not null,
	primary key(id)
);

-- function called by trigger on every insert to events table
-- sends notification on channel, allowing services to subscribe
-- to events when new events are created
create or replace function notify_new_event ()
  returns trigger
  language plpgsql
 as $$
 declare
   channel text := TG_ARGV[0];
 begin
   PERFORM (
      select pg_notify(channel, row_to_json(NEW)::text)
   );
   RETURN NULL;
 end;
 $$;

-- trigger for inserting to events table
-- allows services to LISTEN on 'new_event' in order to get
-- notifications when new events are generated
 CREATE TRIGGER notify_new_event
          AFTER INSERT
             ON events
       FOR EACH ROW
        EXECUTE PROCEDURE notify_new_event('new_event');

 commit;
