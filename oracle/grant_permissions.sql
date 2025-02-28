-- for docker-compose

alter session set container=freepdb1;
alter user pdbadmin identified by "Welcome12345";
grant unlimited tablespace to pdbadmin;
grant select_catalog_role to pdbadmin;
grant execute on dbms_aq to pdbadmin;
grant execute on dbms_aqadm to pdbadmin;
grant execute on dbms_aqin to pdbadmin;
grant execute on dbms_aqjms_internal to pdbadmin;
grant execute on dbms_teqk to pdbadmin;
grant execute on DBMS_RESOURCE_MANAGER to pdbadmin;
grant select on sys.aq$_queue_shards to pdbadmin;
grant select on user_queue_partition_assignment_table to pdbadmin;