-- account table
drop table if exists ftp_account;
create table ftp_account (
    "id" serial primary key,
    username varchar(255) not null,
    description varchar(255) not null,
    "password" varchar(255) not null,
    updated_on timestamp not null default current_timestamp,
    constraint uc_username unique (username)
);

-- mapping table
drop table if exists ftp_mapping;
create table ftp_mapping (
    "system" varchar(255) not null,
    "id" varchar(255) not null,
    ftp_id integer not null,
    primary key ("system", "id"),
    constraint fk_ftp_account foreign key (ftp_id) references ftp_account ("id") on delete cascade
);
