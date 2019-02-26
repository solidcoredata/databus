create database library;
set database = library;

create table book (
	id int,
	name text,
	genre int,
	page_count int
);
create table genre (
	id int,
	name text
);
