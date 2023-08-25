create table if not exists requets(
  id serial primary key,
  created_at timestamp with time zone default now(),
  path varchar(5000)
);