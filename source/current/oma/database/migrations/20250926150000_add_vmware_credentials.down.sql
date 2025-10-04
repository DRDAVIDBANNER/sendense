-- Remove VMware credentials management table and related fields

ALTER TABLE vm_replication_contexts 
DROP FOREIGN KEY fk_vm_context_vmware_creds,
DROP COLUMN vmware_credential_id;

DROP TABLE vmware_credentials;

