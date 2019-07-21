create database library;
set database = library;

create table genre (
	id int not null primary key,
	name string(1000) not null
);
create table book (
	id int not null primary key,
	name string(1000) not null,
	genre int null references genre,
	page_count int null
);
