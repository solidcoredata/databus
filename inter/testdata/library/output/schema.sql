create database library;
set database = library;

create table book (
	id int not null primary key,
	name string(1000) not null,
	genre int null,
	page_count int null
);
create table genre (
	id int not null primary key,
	name string(1000) not null
);
