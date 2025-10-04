/*M!999999\- enable the sandbox mode */ 
-- MariaDB dump 10.19  Distrib 10.11.13-MariaDB, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: migratekit_oma
-- ------------------------------------------------------
-- Server version	10.11.13-MariaDB-0ubuntu0.24.04.1

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Temporary table structure for view `active_jobs`
--

DROP TABLE IF EXISTS `active_jobs`;
/*!50001 DROP VIEW IF EXISTS `active_jobs`*/;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8mb4;
/*!50001 CREATE VIEW `active_jobs` AS SELECT
 1 AS `id`,
  1 AS `job_type`,
  1 AS `operation`,
  1 AS `status`,
  1 AS `parent_job_id`,
  1 AS `started_at`,
  1 AS `running_seconds`,
  1 AS `error_count` */;
SET character_set_client = @saved_cs_client;

--
-- Temporary table structure for view `active_schedules`
--

DROP TABLE IF EXISTS `active_schedules`;
/*!50001 DROP VIEW IF EXISTS `active_schedules`*/;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8mb4;
/*!50001 CREATE VIEW `active_schedules` AS SELECT
 1 AS `id`,
  1 AS `name`,
  1 AS `description`,
  1 AS `cron_expression`,
  1 AS `schedule_type`,
  1 AS `enabled`,
  1 AS `max_concurrent_jobs`,
  1 AS `group_count`,
  1 AS `vm_count`,
  1 AS `next_execution`,
  1 AS `last_execution` */;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `cbt_history`
--

DROP TABLE IF EXISTS `cbt_history`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `cbt_history` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `vm_context_id` varchar(64) DEFAULT NULL,
  `job_id` varchar(191) NOT NULL,
  `disk_id` longtext NOT NULL,
  `change_id` longtext NOT NULL,
  `previous_change_id` longtext DEFAULT NULL,
  `sync_type` longtext NOT NULL,
  `blocks_changed` bigint(20) DEFAULT NULL,
  `bytes_transferred` bigint(20) DEFAULT NULL,
  `sync_duration_seconds` bigint(20) DEFAULT NULL,
  `sync_success` tinyint(1) DEFAULT 0,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_cbt_history_job` (`job_id`),
  KEY `idx_cbt_history_context` (`vm_context_id`),
  CONSTRAINT `fk_cbt_history_context` FOREIGN KEY (`vm_context_id`) REFERENCES `vm_replication_contexts` (`context_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_cbt_history_job` FOREIGN KEY (`job_id`) REFERENCES `replication_jobs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=1463 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `device_mappings`
--

DROP TABLE IF EXISTS `device_mappings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `device_mappings` (
  `id` varchar(64) NOT NULL DEFAULT uuid(),
  `vm_context_id` varchar(64) DEFAULT NULL,
  `volume_uuid` varchar(64) NOT NULL,
  `volume_id_numeric` bigint(20) DEFAULT NULL,
  `vm_id` varchar(64) NOT NULL,
  `operation_mode` enum('oma','failover') DEFAULT 'oma',
  `cloudstack_device_id` int(11) DEFAULT NULL,
  `requires_device_correlation` tinyint(1) DEFAULT 1,
  `device_path` varchar(255) NOT NULL,
  `cloudstack_state` varchar(32) NOT NULL,
  `linux_state` varchar(32) NOT NULL,
  `size` bigint(20) NOT NULL,
  `last_sync` timestamp NULL DEFAULT current_timestamp(),
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `ossea_snapshot_id` varchar(191) DEFAULT NULL COMMENT 'CloudStack volume snapshot ID for rollback protection during test failover',
  `snapshot_created_at` timestamp NULL DEFAULT NULL COMMENT 'Timestamp when snapshot was created during failover operation',
  `snapshot_status` varchar(50) DEFAULT 'none' COMMENT 'Snapshot status: none, creating, ready, failed, rollback_complete',
  `persistent_device_name` varchar(255) DEFAULT NULL COMMENT 'Stable device name for NBD export consistency (e.g., pgtest3disk0)',
  `symlink_path` varchar(255) DEFAULT NULL COMMENT 'Device mapper symlink path for persistent access (e.g., /dev/mapper/pgtest3disk0)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `volume_id` (`volume_uuid`),
  UNIQUE KEY `unique_volume_id` (`volume_uuid`),
  UNIQUE KEY `unique_device_path` (`device_path`),
  KEY `idx_device_mappings_vm_id` (`vm_id`),
  KEY `idx_device_mappings_device_path` (`device_path`),
  KEY `idx_device_mappings_last_sync` (`last_sync`),
  KEY `idx_device_mappings_volume_id` (`volume_id_numeric`),
  KEY `idx_device_mappings_context` (`vm_context_id`),
  KEY `idx_device_mappings_snapshot_id` (`ossea_snapshot_id`),
  KEY `idx_device_mappings_snapshot_status` (`snapshot_status`),
  KEY `idx_device_mappings_vm_context_snapshot` (`vm_context_id`,`snapshot_status`),
  KEY `idx_device_mappings_persistent_name` (`persistent_device_name`),
  KEY `idx_device_mappings_symlink_path` (`symlink_path`),
  CONSTRAINT `fk_device_mappings_context` FOREIGN KEY (`vm_context_id`) REFERENCES `vm_replication_contexts` (`context_id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `failover_jobs`
--

DROP TABLE IF EXISTS `failover_jobs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `failover_jobs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `vm_context_id` varchar(64) DEFAULT NULL,
  `job_id` varchar(255) NOT NULL,
  `vm_id` varchar(255) NOT NULL,
  `replication_job_id` varchar(255) DEFAULT NULL,
  `job_type` enum('live','test') NOT NULL,
  `status` enum('pending','validating','snapshotting','creating_vm','switching_volume','powering_on','completed','failed','cleanup','reverting') NOT NULL DEFAULT 'pending',
  `source_vm_name` varchar(255) NOT NULL,
  `source_vm_spec` text DEFAULT NULL,
  `destination_vm_id` varchar(255) DEFAULT NULL,
  `ossea_snapshot_id` varchar(255) DEFAULT NULL,
  `linstor_snapshot_name` varchar(255) DEFAULT NULL,
  `linstor_config_id` int(11) DEFAULT NULL,
  `network_mappings` text DEFAULT NULL,
  `error_message` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `started_at` timestamp NULL DEFAULT NULL,
  `completed_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `job_id` (`job_id`),
  KEY `idx_vm_id` (`vm_id`),
  KEY `idx_job_type` (`job_type`),
  KEY `idx_status` (`status`),
  KEY `idx_failover_jobs_linstor_snapshot` (`linstor_snapshot_name`),
  KEY `idx_failover_jobs_context` (`vm_context_id`),
  CONSTRAINT `fk_failover_job_context` FOREIGN KEY (`vm_context_id`) REFERENCES `vm_replication_contexts` (`context_id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=214 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `job_execution_log`
--

DROP TABLE IF EXISTS `job_execution_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `job_execution_log` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `job_id` varchar(64) NOT NULL,
  `log_level` varchar(20) NOT NULL,
  `message` text NOT NULL,
  `details` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`details`)),
  `operation_phase` varchar(100) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_job_log_job_id` (`job_id`),
  KEY `idx_job_log_created_at` (`created_at`),
  KEY `idx_job_log_level` (`log_level`),
  KEY `idx_job_log_operation_phase` (`operation_phase`),
  CONSTRAINT `job_execution_log_ibfk_1` FOREIGN KEY (`job_id`) REFERENCES `job_tracking` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=175 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `job_steps`
--

DROP TABLE IF EXISTS `job_steps`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `job_steps` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `job_id` varchar(64) NOT NULL,
  `name` varchar(200) NOT NULL,
  `seq` int(11) NOT NULL,
  `status` enum('running','completed','failed','skipped') NOT NULL DEFAULT 'running',
  `started_at` datetime(6) NOT NULL DEFAULT current_timestamp(6),
  `completed_at` datetime(6) DEFAULT NULL,
  `error_message` longtext DEFAULT NULL,
  `metadata` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`metadata`)),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_job_steps_job_seq` (`job_id`,`seq`),
  CONSTRAINT `job_steps_ibfk_1` FOREIGN KEY (`job_id`) REFERENCES `job_tracking` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=3587 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `job_tracking`
--

DROP TABLE IF EXISTS `job_tracking`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `job_tracking` (
  `id` varchar(64) NOT NULL DEFAULT uuid(),
  `parent_job_id` varchar(64) DEFAULT NULL,
  `job_type` enum('cleanup','failover','migration','cloudstack','volume_daemon','linstor','virtio','ossea','scheduler','discovery','bulk-operations','group-management','conflict-detection','phantom-detection','schedule_management','schedule_control') DEFAULT NULL,
  `operation` varchar(100) NOT NULL,
  `status` enum('pending','running','completed','failed','cancelled') DEFAULT NULL,
  `cloudstack_job_id` varchar(64) DEFAULT NULL,
  `external_job_id` varchar(64) DEFAULT NULL,
  `metadata` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`metadata`)),
  `error_message` longtext DEFAULT NULL,
  `started_at` timestamp NULL DEFAULT current_timestamp(),
  `completed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `percent_complete` tinyint(3) unsigned DEFAULT NULL,
  `canceled_at` datetime(6) DEFAULT NULL,
  `owner` varchar(100) DEFAULT NULL,
  `context_id` varchar(64) DEFAULT NULL COMMENT 'VM context correlation for failover/replication jobs',
  `job_category` enum('system','failover','replication','scheduler','discovery','bulk') DEFAULT 'system' COMMENT 'High-level job categorization for filtering and organization',
  PRIMARY KEY (`id`),
  KEY `idx_parent_job` (`parent_job_id`),
  KEY `idx_status` (`status`),
  KEY `idx_job_type` (`job_type`),
  KEY `idx_cloudstack_job` (`cloudstack_job_id`),
  KEY `idx_external_job` (`external_job_id`),
  KEY `idx_started_at` (`started_at`),
  KEY `idx_completed_at` (`completed_at`),
  KEY `idx_job_tracking_context_id` (`context_id`),
  KEY `idx_job_tracking_category` (`job_category`),
  KEY `idx_job_tracking_category_status_started` (`job_category`,`status`,`started_at`),
  CONSTRAINT `job_tracking_ibfk_1` FOREIGN KEY (`parent_job_id`) REFERENCES `job_tracking` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Enhanced job tracking with VM context correlation and external job ID mapping for GUI integration';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Temporary table structure for view `job_tracking_hierarchy`
--

DROP TABLE IF EXISTS `job_tracking_hierarchy`;
/*!50001 DROP VIEW IF EXISTS `job_tracking_hierarchy`*/;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8mb4;
/*!50001 CREATE VIEW `job_tracking_hierarchy` AS SELECT
 1 AS `id`,
  1 AS `job_type`,
  1 AS `operation`,
  1 AS `status`,
  1 AS `parent_job_id`,
  1 AS `started_at`,
  1 AS `completed_at`,
  1 AS `job_level`,
  1 AS `child_count`,
  1 AS `duration_seconds` */;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `linstor_configs`
--

DROP TABLE IF EXISTS `linstor_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `linstor_configs` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `api_url` varchar(255) NOT NULL,
  `api_port` int(11) DEFAULT 3370,
  `api_protocol` varchar(10) DEFAULT 'http',
  `api_key` varchar(512) DEFAULT NULL,
  `api_secret` varchar(512) DEFAULT NULL,
  `connection_timeout_seconds` int(11) DEFAULT 30,
  `retry_attempts` int(11) DEFAULT 3,
  `description` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `is_active` tinyint(1) DEFAULT 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`),
  KEY `idx_linstor_configs_active` (`is_active`),
  KEY `idx_linstor_configs_name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `log_events`
--

DROP TABLE IF EXISTS `log_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `log_events` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `job_id` varchar(64) DEFAULT NULL,
  `step_id` bigint(20) DEFAULT NULL,
  `level` enum('DEBUG','INFO','WARN','ERROR') NOT NULL,
  `message` text NOT NULL,
  `attrs` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`attrs`)),
  `ts` datetime(6) NOT NULL DEFAULT current_timestamp(6),
  `external_job_id` varchar(255) DEFAULT NULL COMMENT 'GUI-constructed job ID for fast correlation (e.g., unified-live-failover-pgtest2-1758553933)',
  PRIMARY KEY (`id`),
  KEY `job_id` (`job_id`),
  KEY `step_id` (`step_id`),
  KEY `idx_log_events_external_job_id` (`external_job_id`),
  CONSTRAINT `log_events_ibfk_1` FOREIGN KEY (`job_id`) REFERENCES `job_tracking` (`id`) ON DELETE SET NULL,
  CONSTRAINT `log_events_ibfk_2` FOREIGN KEY (`step_id`) REFERENCES `job_steps` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB AUTO_INCREMENT=36699 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Structured log events with job correlation and GUI job ID mapping for progress tracking';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `nbd_exports`
--

DROP TABLE IF EXISTS `nbd_exports`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `nbd_exports` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `vm_context_id` varchar(64) DEFAULT NULL,
  `job_id` varchar(191) DEFAULT NULL,
  `volume_id` varchar(64) DEFAULT '',
  `vm_disk_id` bigint(20) DEFAULT NULL,
  `device_mapping_uuid` varchar(64) DEFAULT NULL,
  `export_name` varchar(191) NOT NULL,
  `port` bigint(20) NOT NULL,
  `device_path` longtext NOT NULL,
  `config_path` longtext DEFAULT '',
  `status` varchar(191) NOT NULL DEFAULT 'pending',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_nbd_exports_export_name` (`export_name`),
  KEY `idx_nbd_exports_job_id` (`job_id`),
  KEY `idx_nbd_exports_volume_id` (`volume_id`),
  KEY `fk_nbd_vm_disk` (`vm_disk_id`),
  KEY `fk_nbd_device` (`device_mapping_uuid`),
  KEY `idx_nbd_exports_status` (`status`),
  KEY `idx_nbd_exports_context` (`vm_context_id`),
  CONSTRAINT `fk_nbd_device` FOREIGN KEY (`device_mapping_uuid`) REFERENCES `device_mappings` (`volume_uuid`) ON DELETE CASCADE,
  CONSTRAINT `fk_nbd_exports_context` FOREIGN KEY (`vm_context_id`) REFERENCES `vm_replication_contexts` (`context_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_nbd_job` FOREIGN KEY (`job_id`) REFERENCES `replication_jobs` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_nbd_vm_disk` FOREIGN KEY (`vm_disk_id`) REFERENCES `vm_disks` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=437 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `network_mappings`
--

DROP TABLE IF EXISTS `network_mappings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `network_mappings` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `vm_id` varchar(255) NOT NULL,
  `source_network_name` varchar(255) NOT NULL,
  `destination_network_id` varchar(255) NOT NULL,
  `destination_network_name` varchar(255) NOT NULL,
  `is_test_network` tinyint(1) DEFAULT 0,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `vm_context_id` varchar(64) DEFAULT NULL,
  `vmware_vm_id` varchar(255) DEFAULT NULL,
  `validation_status` enum('pending','valid','invalid') DEFAULT 'pending',
  `mapping_type` enum('live','test','both') DEFAULT 'live',
  `network_strategy` varchar(50) DEFAULT NULL,
  `last_validated` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_vm_network_type` (`vm_id`,`source_network_name`,`is_test_network`),
  UNIQUE KEY `unique_context_network_type` (`vm_context_id`,`source_network_name`,`is_test_network`),
  KEY `idx_vm_id` (`vm_id`),
  KEY `idx_is_test` (`is_test_network`),
  KEY `idx_network_mappings_vm_context_id` (`vm_context_id`),
  KEY `idx_network_mappings_vmware_vm_id` (`vmware_vm_id`),
  KEY `idx_network_mappings_validation_status` (`validation_status`),
  KEY `idx_network_mappings_mapping_type` (`mapping_type`),
  CONSTRAINT `fk_network_mappings_vm_context` FOREIGN KEY (`vm_context_id`) REFERENCES `vm_replication_contexts` (`context_id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=45 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `ossea_configs`
--

DROP TABLE IF EXISTS `ossea_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `ossea_configs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` longtext NOT NULL,
  `api_url` longtext NOT NULL,
  `api_key` longtext NOT NULL,
  `secret_key` longtext NOT NULL,
  `domain` longtext DEFAULT NULL,
  `zone` longtext NOT NULL,
  `template_id` longtext DEFAULT NULL,
  `network_id` longtext DEFAULT NULL,
  `service_offering_id` longtext DEFAULT NULL,
  `disk_offering_id` longtext DEFAULT NULL,
  `oma_vm_id` longtext DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `is_active` tinyint(1) DEFAULT 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ossea_configs_name` (`name`) USING HASH
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `ossea_volumes`
--

DROP TABLE IF EXISTS `ossea_volumes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `ossea_volumes` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `vm_context_id` varchar(64) DEFAULT NULL,
  `volume_id` longtext NOT NULL,
  `volume_name` longtext NOT NULL,
  `size_gb` bigint(20) NOT NULL,
  `ossea_config_id` bigint(20) DEFAULT NULL,
  `volume_type` longtext DEFAULT NULL,
  `device_path` longtext DEFAULT NULL,
  `mount_point` longtext DEFAULT NULL,
  `status` varchar(191) DEFAULT 'creating',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `snapshot_id` varchar(191) DEFAULT NULL COMMENT 'CloudStack volume snapshot ID for test failover protection',
  `snapshot_created_at` timestamp NULL DEFAULT NULL COMMENT 'Timestamp when snapshot was created during test failover',
  `snapshot_status` varchar(50) DEFAULT 'none' COMMENT 'Snapshot status: none, creating, ready, failed, rollback_complete',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ossea_volumes_volume_id` (`volume_id`) USING HASH,
  KEY `fk_ossea_volumes_ossea_config` (`ossea_config_id`),
  KEY `idx_ossea_volumes_context` (`vm_context_id`),
  KEY `idx_ossea_volumes_snapshot_id` (`snapshot_id`),
  KEY `idx_ossea_volumes_snapshot_status` (`snapshot_status`),
  CONSTRAINT `fk_ossea_volumes_context` FOREIGN KEY (`vm_context_id`) REFERENCES `vm_replication_contexts` (`context_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_ossea_volumes_ossea_config` FOREIGN KEY (`ossea_config_id`) REFERENCES `ossea_configs` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=199 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `replication_jobs`
--

DROP TABLE IF EXISTS `replication_jobs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `replication_jobs` (
  `id` varchar(191) NOT NULL,
  `vm_context_id` varchar(64) DEFAULT NULL,
  `source_vm_id` longtext NOT NULL,
  `source_vm_name` longtext NOT NULL,
  `source_vm_path` longtext NOT NULL,
  `v_center_host` longtext NOT NULL,
  `datacenter` longtext NOT NULL,
  `replication_type` longtext NOT NULL,
  `target_network` longtext DEFAULT NULL,
  `status` varchar(191) DEFAULT 'pending',
  `progress_percent` double DEFAULT 0,
  `current_operation` longtext DEFAULT NULL,
  `bytes_transferred` bigint(20) DEFAULT 0,
  `total_bytes` bigint(20) DEFAULT 0,
  `transfer_speed_bps` bigint(20) DEFAULT 0,
  `error_message` longtext DEFAULT NULL,
  `change_id` longtext DEFAULT NULL,
  `previous_change_id` longtext DEFAULT NULL,
  `snapshot_id` longtext DEFAULT NULL,
  `nbd_port` bigint(20) DEFAULT NULL,
  `nbd_export_name` longtext DEFAULT NULL,
  `target_device` longtext DEFAULT NULL,
  `ossea_config_id` bigint(20) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `started_at` datetime(3) DEFAULT NULL,
  `completed_at` datetime(3) DEFAULT NULL,
  `vma_sync_type` varchar(50) DEFAULT NULL COMMENT 'VMA detected sync type (initial/incremental)',
  `vma_current_phase` varchar(100) DEFAULT NULL COMMENT 'VMA current phase (Initializing/Snapshot Creation/Copying Data/Cleanup)',
  `vma_throughput_mbps` decimal(10,2) DEFAULT 0.00 COMMENT 'VMA throughput in MB/s',
  `vma_eta_seconds` int(11) DEFAULT NULL COMMENT 'VMA estimated time to completion in seconds',
  `vma_last_poll_at` timestamp NULL DEFAULT NULL COMMENT 'Last time VMA progress was polled',
  `vma_error_classification` varchar(50) DEFAULT NULL COMMENT 'VMA error classification (connection/authentication/permission/disk/network/system)',
  `vma_error_details` text DEFAULT NULL COMMENT 'VMA detailed error information',
  `setup_progress_percent` decimal(5,2) DEFAULT 0.00 COMMENT 'OMA setup progress (0-85%): job creation, volume provisioning, NBD setup',
  `schedule_execution_id` varchar(64) DEFAULT NULL COMMENT 'Links job to schedule execution that created it',
  `scheduled_by` varchar(255) DEFAULT NULL COMMENT 'Which scheduler component created this job',
  `vm_group_id` varchar(64) DEFAULT NULL COMMENT 'Machine group this job belongs to',
  PRIMARY KEY (`id`),
  KEY `fk_replication_jobs_ossea_config` (`ossea_config_id`),
  KEY `idx_replication_jobs_nbd_polling` (`status`,`nbd_export_name`(50)),
  KEY `idx_replication_jobs_progress_timeout` (`status`,`updated_at`),
  KEY `idx_replication_jobs_completion_tracking` (`status`,`completed_at`),
  KEY `idx_replication_jobs_context` (`vm_context_id`),
  KEY `idx_replication_jobs_schedule_execution` (`schedule_execution_id`),
  KEY `idx_replication_jobs_scheduled_by` (`scheduled_by`),
  KEY `idx_replication_jobs_vm_group` (`vm_group_id`),
  CONSTRAINT `fk_replication_job_context` FOREIGN KEY (`vm_context_id`) REFERENCES `vm_replication_contexts` (`context_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_replication_jobs_ossea_config` FOREIGN KEY (`ossea_config_id`) REFERENCES `ossea_configs` (`id`),
  CONSTRAINT `fk_replication_jobs_schedule_execution` FOREIGN KEY (`schedule_execution_id`) REFERENCES `schedule_executions` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_replication_jobs_vm_group` FOREIGN KEY (`vm_group_id`) REFERENCES `vm_machine_groups` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `replication_schedules`
--

DROP TABLE IF EXISTS `replication_schedules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `replication_schedules` (
  `id` varchar(64) NOT NULL DEFAULT uuid(),
  `name` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `cron_expression` varchar(100) NOT NULL COMMENT 'Cron expression for schedule timing (e.g., "0 2 * * *" for daily 2 AM)',
  `schedule_type` enum('cron','chain') NOT NULL DEFAULT 'cron',
  `timezone` varchar(50) DEFAULT 'UTC' COMMENT 'Timezone for cron expression',
  `chain_parent_schedule_id` varchar(64) DEFAULT NULL COMMENT 'Parent schedule for chain dependency',
  `chain_delay_minutes` int(11) DEFAULT 0 COMMENT 'Minutes to wait after parent completion',
  `replication_type` enum('full','incremental','auto') DEFAULT 'auto' COMMENT 'Type of replication for scheduled jobs',
  `max_concurrent_jobs` int(11) DEFAULT 1 COMMENT 'Maximum number of concurrent jobs from this schedule',
  `retry_attempts` int(11) DEFAULT 3 COMMENT 'Number of retry attempts for failed jobs',
  `retry_delay_minutes` int(11) DEFAULT 30 COMMENT 'Minutes between retry attempts',
  `enabled` tinyint(1) DEFAULT 1 COMMENT 'Whether this schedule is active',
  `skip_if_running` tinyint(1) DEFAULT 1 COMMENT 'Skip execution if jobs from this schedule are still running',
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `created_by` varchar(255) DEFAULT 'system',
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`),
  KEY `idx_replication_schedules_enabled` (`enabled`),
  KEY `idx_replication_schedules_name` (`name`),
  KEY `idx_replication_schedules_type` (`schedule_type`),
  KEY `idx_replication_schedules_parent` (`chain_parent_schedule_id`),
  CONSTRAINT `replication_schedules_ibfk_1` FOREIGN KEY (`chain_parent_schedule_id`) REFERENCES `replication_schedules` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Temporary table structure for view `schedule_execution_summary`
--

DROP TABLE IF EXISTS `schedule_execution_summary`;
/*!50001 DROP VIEW IF EXISTS `schedule_execution_summary`*/;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8mb4;
/*!50001 CREATE VIEW `schedule_execution_summary` AS SELECT
 1 AS `id`,
  1 AS `schedule_id`,
  1 AS `schedule_name`,
  1 AS `status`,
  1 AS `scheduled_at`,
  1 AS `started_at`,
  1 AS `completed_at`,
  1 AS `execution_duration_seconds`,
  1 AS `vms_eligible`,
  1 AS `jobs_created`,
  1 AS `jobs_completed`,
  1 AS `jobs_failed`,
  1 AS `jobs_skipped`,
  1 AS `success_rate_percent`,
  1 AS `error_message`,
  1 AS `triggered_by` */;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `schedule_executions`
--

DROP TABLE IF EXISTS `schedule_executions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `schedule_executions` (
  `id` varchar(64) NOT NULL DEFAULT uuid(),
  `schedule_id` varchar(64) NOT NULL,
  `group_id` varchar(64) DEFAULT NULL COMMENT 'Group being processed (null for VM-specific schedules)',
  `scheduled_at` timestamp NOT NULL COMMENT 'When this execution was scheduled to run',
  `started_at` timestamp NULL DEFAULT NULL COMMENT 'When execution actually started',
  `completed_at` timestamp NULL DEFAULT NULL COMMENT 'When execution finished (success or failure)',
  `status` enum('scheduled','running','completed','failed','skipped','cancelled') DEFAULT 'scheduled',
  `vms_eligible` int(11) DEFAULT 0 COMMENT 'Number of VMs eligible for replication',
  `jobs_created` int(11) DEFAULT 0 COMMENT 'Number of replication jobs created',
  `jobs_completed` int(11) DEFAULT 0 COMMENT 'Number of jobs completed successfully',
  `jobs_failed` int(11) DEFAULT 0 COMMENT 'Number of jobs that failed',
  `jobs_skipped` int(11) DEFAULT 0 COMMENT 'Number of VMs skipped (already running)',
  `execution_details` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT 'Detailed execution information (job IDs, VM states, etc.)' CHECK (json_valid(`execution_details`)),
  `error_message` text DEFAULT NULL COMMENT 'Error message if execution failed',
  `error_details` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT 'Detailed error information' CHECK (json_valid(`error_details`)),
  `execution_duration_seconds` int(11) DEFAULT NULL COMMENT 'Total execution time',
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `triggered_by` varchar(255) DEFAULT 'scheduler' COMMENT 'What triggered this execution (scheduler, manual, chain)',
  PRIMARY KEY (`id`),
  KEY `idx_schedule_executions_schedule` (`schedule_id`),
  KEY `idx_schedule_executions_status` (`status`),
  KEY `idx_schedule_executions_scheduled_at` (`scheduled_at`),
  KEY `idx_schedule_executions_started_at` (`started_at`),
  KEY `idx_schedule_executions_group` (`group_id`),
  CONSTRAINT `schedule_executions_ibfk_1` FOREIGN KEY (`schedule_id`) REFERENCES `replication_schedules` (`id`) ON DELETE CASCADE,
  CONSTRAINT `schedule_executions_ibfk_2` FOREIGN KEY (`group_id`) REFERENCES `vm_machine_groups` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `schema_migrations`
--

DROP TABLE IF EXISTS `schema_migrations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `schema_migrations` (
  `version` varchar(14) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `applied_at` timestamp NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vm_disks`
--

DROP TABLE IF EXISTS `vm_disks`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vm_disks` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `vm_context_id` varchar(64) DEFAULT NULL,
  `job_id` varchar(191) NOT NULL,
  `disk_id` longtext NOT NULL,
  `vm_dk_path` longtext NOT NULL,
  `size_gb` bigint(20) NOT NULL,
  `datastore` longtext DEFAULT NULL,
  `unit_number` bigint(20) DEFAULT NULL,
  `label` longtext DEFAULT NULL,
  `capacity_bytes` bigint(20) DEFAULT NULL,
  `provisioning_type` longtext DEFAULT NULL,
  `ossea_volume_id` bigint(20) DEFAULT NULL,
  `disk_change_id` longtext DEFAULT NULL,
  `sync_status` varchar(191) DEFAULT 'pending',
  `sync_progress_percent` double DEFAULT 0,
  `bytes_synced` bigint(20) DEFAULT 0,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `cpu_count` int(11) DEFAULT 0,
  `memory_mb` int(11) DEFAULT 0,
  `os_type` varchar(255) DEFAULT '',
  `vm_tools_version` varchar(255) DEFAULT '',
  `network_config` text DEFAULT NULL,
  `display_name` varchar(255) DEFAULT '',
  `annotation` text DEFAULT NULL,
  `power_state` varchar(50) DEFAULT '',
  `vmware_uuid` varchar(255) DEFAULT '',
  `bios_setup` text DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_vm_context_disk` (`vm_context_id`,`disk_id`) USING HASH,
  KEY `fk_vm_disks_job` (`job_id`),
  KEY `idx_vm_disks_context` (`vm_context_id`),
  KEY `idx_vm_disks_ossea_volume` (`ossea_volume_id`),
  CONSTRAINT `fk_vm_disks_context` FOREIGN KEY (`vm_context_id`) REFERENCES `vm_replication_contexts` (`context_id`) ON DELETE CASCADE,
  CONSTRAINT `fk_vm_disks_job` FOREIGN KEY (`job_id`) REFERENCES `replication_jobs` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=791 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vm_export_mappings`
--

DROP TABLE IF EXISTS `vm_export_mappings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vm_export_mappings` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `vm_id` varchar(36) NOT NULL COMMENT 'VMware VM UUID',
  `disk_unit_number` int(11) NOT NULL COMMENT 'SCSI unit number (0,1,2...)',
  `vm_name` varchar(255) NOT NULL COMMENT 'VMware VM name for reference',
  `export_name` varchar(255) NOT NULL COMMENT 'NBD export name (migration-vm-{id}-disk{unit})',
  `device_path` varchar(255) NOT NULL COMMENT 'Linux device path (/dev/vdb, /dev/vdc, etc.)',
  `status` enum('active','inactive') DEFAULT 'active' COMMENT 'Export availability status',
  `created_at` datetime(3) DEFAULT current_timestamp(3),
  `updated_at` datetime(3) DEFAULT current_timestamp(3) ON UPDATE current_timestamp(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `export_name` (`export_name`),
  UNIQUE KEY `unique_vm_disk` (`vm_id`,`disk_unit_number`),
  UNIQUE KEY `unique_device_path` (`device_path`),
  KEY `idx_vm_id` (`vm_id`),
  KEY `idx_export_name` (`export_name`),
  KEY `idx_device_path` (`device_path`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=30 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Persistent mapping between VMware VMs and NBD exports to enable export reuse';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vm_group_memberships`
--

DROP TABLE IF EXISTS `vm_group_memberships`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vm_group_memberships` (
  `id` varchar(64) NOT NULL DEFAULT uuid(),
  `group_id` varchar(64) NOT NULL,
  `vm_context_id` varchar(64) NOT NULL COMMENT 'References vm_replication_contexts.context_id',
  `enabled` tinyint(1) DEFAULT 1 COMMENT 'Whether this VM participates in scheduled replications',
  `priority` int(11) DEFAULT 0 COMMENT 'VM priority within group (lower number = higher priority)',
  `schedule_override_id` varchar(64) DEFAULT NULL COMMENT 'Optional override schedule for this specific VM',
  `added_at` timestamp NULL DEFAULT current_timestamp(),
  `added_by` varchar(255) DEFAULT 'system',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_vm_group` (`group_id`,`vm_context_id`),
  KEY `schedule_override_id` (`schedule_override_id`),
  KEY `idx_vm_group_memberships_group` (`group_id`),
  KEY `idx_vm_group_memberships_vm` (`vm_context_id`),
  KEY `idx_vm_group_memberships_enabled` (`enabled`),
  KEY `idx_vm_group_memberships_priority` (`priority`),
  CONSTRAINT `vm_group_memberships_ibfk_1` FOREIGN KEY (`group_id`) REFERENCES `vm_machine_groups` (`id`) ON DELETE CASCADE,
  CONSTRAINT `vm_group_memberships_ibfk_2` FOREIGN KEY (`vm_context_id`) REFERENCES `vm_replication_contexts` (`context_id`) ON DELETE CASCADE,
  CONSTRAINT `vm_group_memberships_ibfk_3` FOREIGN KEY (`schedule_override_id`) REFERENCES `replication_schedules` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vm_machine_groups`
--

DROP TABLE IF EXISTS `vm_machine_groups`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vm_machine_groups` (
  `id` varchar(64) NOT NULL DEFAULT uuid(),
  `name` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `schedule_id` varchar(64) DEFAULT NULL COMMENT 'Default schedule for this group',
  `max_concurrent_vms` int(11) DEFAULT 5 COMMENT 'Maximum VMs to process concurrently in this group',
  `priority` int(11) DEFAULT 0 COMMENT 'Group priority (lower number = higher priority)',
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `created_by` varchar(255) DEFAULT 'system',
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`),
  KEY `idx_vm_machine_groups_name` (`name`),
  KEY `idx_vm_machine_groups_schedule` (`schedule_id`),
  KEY `idx_vm_machine_groups_priority` (`priority`),
  CONSTRAINT `vm_machine_groups_ibfk_1` FOREIGN KEY (`schedule_id`) REFERENCES `replication_schedules` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vm_replication_contexts`
--

DROP TABLE IF EXISTS `vm_replication_contexts`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vm_replication_contexts` (
  `context_id` varchar(64) NOT NULL DEFAULT uuid(),
  `vm_name` varchar(255) NOT NULL,
  `vmware_vm_id` varchar(255) NOT NULL,
  `vm_path` varchar(500) NOT NULL,
  `vcenter_host` varchar(255) NOT NULL,
  `datacenter` varchar(255) NOT NULL,
  `current_status` enum('discovered','replicating','ready_for_failover','failed_over_test','failed_over_live','completed','failed','cleanup_required') NOT NULL DEFAULT 'discovered',
  `current_job_id` varchar(191) DEFAULT NULL,
  `total_jobs_run` int(11) DEFAULT 0,
  `successful_jobs` int(11) DEFAULT 0,
  `failed_jobs` int(11) DEFAULT 0,
  `last_successful_job_id` varchar(191) DEFAULT NULL,
  `cpu_count` int(11) DEFAULT NULL,
  `memory_mb` int(11) DEFAULT NULL,
  `os_type` varchar(255) DEFAULT NULL,
  `power_state` varchar(50) DEFAULT NULL,
  `vm_tools_version` varchar(255) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `first_job_at` timestamp NULL DEFAULT NULL,
  `last_job_at` timestamp NULL DEFAULT NULL,
  `last_status_change` timestamp NULL DEFAULT current_timestamp(),
  `auto_added` tinyint(1) DEFAULT 0 COMMENT 'VM was added via discovery without immediate replication job',
  `last_scheduled_job_id` varchar(255) DEFAULT NULL COMMENT 'Most recent job created by scheduler',
  `next_scheduled_at` timestamp NULL DEFAULT NULL COMMENT 'When this VM is next scheduled for replication',
  `scheduler_enabled` tinyint(1) DEFAULT 1 COMMENT 'Whether this VM participates in scheduled replications',
  `credential_id` int(11) DEFAULT NULL,
  `ossea_config_id` int(11) DEFAULT NULL,
  `last_operation_summary` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT 'Summary of most recent operation (replication/failover/rollback) for GUI visibility' CHECK (json_valid(`last_operation_summary`)),
  PRIMARY KEY (`context_id`),
  UNIQUE KEY `unique_vm_per_vcenter` (`vmware_vm_id`,`vcenter_host`),
  UNIQUE KEY `unique_vm_name_per_vcenter` (`vm_name`,`vcenter_host`),
  KEY `fk_vm_context_last_job` (`last_successful_job_id`),
  KEY `idx_vm_name` (`vm_name`),
  KEY `idx_vmware_id` (`vmware_vm_id`),
  KEY `idx_current_status` (`current_status`),
  KEY `idx_current_job` (`current_job_id`),
  KEY `idx_vcenter` (`vcenter_host`),
  KEY `idx_vm_contexts_auto_added` (`auto_added`),
  KEY `idx_vm_contexts_next_scheduled` (`next_scheduled_at`),
  KEY `idx_vm_contexts_scheduler_enabled` (`scheduler_enabled`),
  KEY `fk_vm_context_vmware_creds` (`credential_id`),
  CONSTRAINT `fk_vm_context_current_job` FOREIGN KEY (`current_job_id`) REFERENCES `replication_jobs` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_vm_context_last_job` FOREIGN KEY (`last_successful_job_id`) REFERENCES `replication_jobs` (`id`) ON DELETE SET NULL,
  CONSTRAINT `fk_vm_context_vmware_creds` FOREIGN KEY (`credential_id`) REFERENCES `vmware_credentials` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Temporary table structure for view `vm_schedule_status`
--

DROP TABLE IF EXISTS `vm_schedule_status`;
/*!50001 DROP VIEW IF EXISTS `vm_schedule_status`*/;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8mb4;
/*!50001 CREATE VIEW `vm_schedule_status` AS SELECT
 1 AS `context_id`,
  1 AS `vm_name`,
  1 AS `vm_status`,
  1 AS `next_scheduled_at`,
  1 AS `scheduler_enabled`,
  1 AS `group_name`,
  1 AS `group_priority`,
  1 AS `vm_priority`,
  1 AS `membership_enabled`,
  1 AS `schedule_name`,
  1 AS `cron_expression`,
  1 AS `schedule_enabled`,
  1 AS `has_active_job`,
  1 AS `active_job_id`,
  1 AS `job_status`,
  1 AS `progress_percent` */;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `vma_active_connections`
--

DROP TABLE IF EXISTS `vma_active_connections`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vma_active_connections` (
  `id` varchar(36) NOT NULL DEFAULT uuid(),
  `enrollment_id` varchar(36) NOT NULL COMMENT 'Reference to approved enrollment',
  `vma_name` varchar(255) NOT NULL COMMENT 'VMA identifier',
  `vma_fingerprint` varchar(255) NOT NULL COMMENT 'SSH key fingerprint',
  `ssh_user` varchar(50) NOT NULL DEFAULT 'vma_tunnel' COMMENT 'SSH user for tunnel connection',
  `connection_status` enum('connected','disconnected','revoked') NOT NULL DEFAULT 'connected' COMMENT 'Current connection status',
  `last_seen_at` timestamp NULL DEFAULT NULL COMMENT 'Last successful health check',
  `connected_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `revoked_at` timestamp NULL DEFAULT NULL COMMENT 'When access was revoked',
  `revoked_by` varchar(255) DEFAULT NULL COMMENT 'Admin who revoked access',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_vma_connection` (`enrollment_id`),
  KEY `idx_vma_connections_status` (`connection_status`),
  KEY `idx_vma_connections_last_seen` (`last_seen_at`),
  KEY `idx_vma_connections_fingerprint` (`vma_fingerprint`),
  CONSTRAINT `fk_vma_connection_enrollment` FOREIGN KEY (`enrollment_id`) REFERENCES `vma_enrollments` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Active VMA tunnel connections for monitoring and management';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vma_connection_audit`
--

DROP TABLE IF EXISTS `vma_connection_audit`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vma_connection_audit` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `enrollment_id` varchar(36) DEFAULT NULL COMMENT 'Reference to vma_enrollments.id',
  `event_type` enum('enrollment','verification','approval','rejection','connection','disconnection','revocation') NOT NULL COMMENT 'Type of security event',
  `vma_fingerprint` varchar(255) DEFAULT NULL COMMENT 'SSH key fingerprint for correlation',
  `source_ip` varchar(45) DEFAULT NULL COMMENT 'Source IP address of event',
  `user_agent` varchar(255) DEFAULT NULL COMMENT 'User agent or client identifier',
  `approved_by` varchar(255) DEFAULT NULL COMMENT 'Admin user for approval/rejection events',
  `event_details` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT 'Additional event metadata' CHECK (json_valid(`event_details`)),
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `idx_vma_audit_enrollment_id` (`enrollment_id`),
  KEY `idx_vma_audit_event_type` (`event_type`),
  KEY `idx_vma_audit_created_at` (`created_at`),
  KEY `idx_vma_audit_vma_fingerprint` (`vma_fingerprint`),
  KEY `idx_vma_audit_source_ip` (`source_ip`),
  CONSTRAINT `fk_vma_audit_enrollment` FOREIGN KEY (`enrollment_id`) REFERENCES `vma_enrollments` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Complete audit trail for VMA enrollment and connection events';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vma_enrollments`
--

DROP TABLE IF EXISTS `vma_enrollments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vma_enrollments` (
  `id` varchar(36) NOT NULL DEFAULT uuid(),
  `pairing_code` varchar(20) NOT NULL COMMENT 'Short-lived pairing code (AX7K-PJ3F-TH2Q format)',
  `vma_public_key` text NOT NULL COMMENT 'Ed25519 public key from VMA',
  `vma_name` varchar(255) DEFAULT NULL COMMENT 'Human-readable VMA identifier',
  `vma_version` varchar(100) DEFAULT NULL COMMENT 'VMA software version',
  `vma_fingerprint` varchar(255) DEFAULT NULL COMMENT 'SSH key fingerprint for display',
  `vma_ip_address` varchar(45) DEFAULT NULL COMMENT 'Source IP address of enrollment request',
  `challenge_nonce` varchar(64) DEFAULT NULL COMMENT 'Cryptographic challenge for key verification',
  `status` enum('pending_verification','awaiting_approval','approved','rejected','expired') NOT NULL DEFAULT 'pending_verification' COMMENT 'Enrollment workflow status',
  `approved_by` varchar(255) DEFAULT NULL COMMENT 'Admin user who approved this enrollment',
  `approved_at` timestamp NULL DEFAULT NULL COMMENT 'When enrollment was approved',
  `expires_at` timestamp NOT NULL COMMENT 'When pairing code expires',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `pairing_code` (`pairing_code`),
  KEY `idx_vma_enrollments_pairing_code` (`pairing_code`),
  KEY `idx_vma_enrollments_status` (`status`),
  KEY `idx_vma_enrollments_expires_at` (`expires_at`),
  KEY `idx_vma_enrollments_created_at` (`created_at`),
  KEY `idx_vma_enrollments_approved_by` (`approved_by`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='VMA enrollment requests with operator approval workflow';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vma_pairing_codes`
--

DROP TABLE IF EXISTS `vma_pairing_codes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vma_pairing_codes` (
  `id` varchar(36) NOT NULL DEFAULT uuid(),
  `pairing_code` varchar(20) NOT NULL COMMENT 'Generated pairing code',
  `generated_by` varchar(255) NOT NULL COMMENT 'Admin who generated the code',
  `used_by_enrollment_id` varchar(36) DEFAULT NULL COMMENT 'Which enrollment used this code',
  `expires_at` timestamp NOT NULL COMMENT 'Code expiry time',
  `used_at` timestamp NULL DEFAULT NULL COMMENT 'When code was used',
  `created_at` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`id`),
  UNIQUE KEY `pairing_code` (`pairing_code`),
  UNIQUE KEY `unique_pairing_code` (`pairing_code`),
  KEY `idx_pairing_codes_expires_at` (`expires_at`),
  KEY `idx_pairing_codes_generated_by` (`generated_by`),
  KEY `idx_pairing_codes_used_at` (`used_at`),
  KEY `fk_pairing_code_enrollment` (`used_by_enrollment_id`),
  CONSTRAINT `fk_pairing_code_enrollment` FOREIGN KEY (`used_by_enrollment_id`) REFERENCES `vma_enrollments` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Pairing code generation and usage tracking for security audit';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `vmware_credentials`
--

DROP TABLE IF EXISTS `vmware_credentials`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `vmware_credentials` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `credential_name` varchar(255) NOT NULL COMMENT 'Human-readable name (e.g., Production-vCenter, Dev-vCenter)',
  `vcenter_host` varchar(255) NOT NULL COMMENT 'vCenter hostname or IP address',
  `username` varchar(255) NOT NULL COMMENT 'vCenter username (e.g., administrator@vsphere.local)',
  `password_encrypted` text NOT NULL COMMENT 'AES-256 encrypted password',
  `datacenter` varchar(255) NOT NULL COMMENT 'Default datacenter name for this vCenter',
  `is_active` tinyint(1) DEFAULT 1 COMMENT 'Enable/disable this credential set',
  `is_default` tinyint(1) DEFAULT 0 COMMENT 'Default credential set for operations',
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `created_by` varchar(255) DEFAULT NULL COMMENT 'User who created this credential set',
  `last_used` timestamp NULL DEFAULT NULL COMMENT 'Last time these credentials were used in operations',
  `usage_count` int(11) DEFAULT 0 COMMENT 'Number of times credentials have been used',
  PRIMARY KEY (`id`),
  UNIQUE KEY `credential_name` (`credential_name`),
  KEY `idx_vmware_creds_active` (`is_active`),
  KEY `idx_vmware_creds_default` (`is_default`),
  KEY `idx_vmware_creds_host` (`vcenter_host`),
  KEY `idx_vmware_creds_last_used` (`last_used`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Centralized VMware vCenter credential management';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `volume_daemon_metrics`
--

DROP TABLE IF EXISTS `volume_daemon_metrics`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `volume_daemon_metrics` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `timestamp` timestamp NULL DEFAULT current_timestamp(),
  `total_operations` bigint(20) NOT NULL DEFAULT 0,
  `pending_operations` bigint(20) NOT NULL DEFAULT 0,
  `active_mappings` bigint(20) NOT NULL DEFAULT 0,
  `operations_by_type` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`operations_by_type`)),
  `operations_by_status` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`operations_by_status`)),
  `average_response_time_ms` decimal(10,2) NOT NULL DEFAULT 0.00,
  `error_rate_percent` decimal(5,2) NOT NULL DEFAULT 0.00,
  `details` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`details`)),
  PRIMARY KEY (`id`),
  KEY `idx_daemon_metrics_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `volume_mounts`
--

DROP TABLE IF EXISTS `volume_mounts`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `volume_mounts` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `ossea_volume_id` bigint(20) NOT NULL,
  `job_id` longtext NOT NULL,
  `device_path` longtext NOT NULL,
  `mount_point` longtext DEFAULT NULL,
  `mount_status` varchar(191) DEFAULT 'unmounted',
  `filesystem_type` longtext DEFAULT NULL,
  `mount_options` longtext DEFAULT NULL,
  `is_read_only` tinyint(1) DEFAULT 0,
  `mounted_at` datetime(3) DEFAULT NULL,
  `unmounted_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `volume_operation_history`
--

DROP TABLE IF EXISTS `volume_operation_history`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `volume_operation_history` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `operation_id` varchar(64) NOT NULL,
  `previous_status` enum('pending','executing','completed','failed','cancelled') DEFAULT NULL,
  `new_status` enum('pending','executing','completed','failed','cancelled') DEFAULT NULL,
  `changed_at` timestamp NULL DEFAULT current_timestamp(),
  `details` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`details`)),
  PRIMARY KEY (`id`),
  KEY `idx_operation_history_operation_id` (`operation_id`),
  KEY `idx_operation_history_changed_at` (`changed_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `volume_operations`
--

DROP TABLE IF EXISTS `volume_operations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `volume_operations` (
  `id` varchar(64) NOT NULL,
  `type` enum('create','attach','detach','delete','cleanup') NOT NULL,
  `status` enum('pending','executing','completed','failed','cancelled') NOT NULL DEFAULT 'pending',
  `volume_id` varchar(64) NOT NULL,
  `vm_id` varchar(64) DEFAULT NULL,
  `request` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`request`)),
  `response` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`response`)),
  `error` text DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT current_timestamp(),
  `updated_at` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `completed_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_volume_operations_volume_id` (`volume_id`),
  KEY `idx_volume_operations_vm_id` (`vm_id`),
  KEY `idx_volume_operations_status` (`status`),
  KEY `idx_volume_operations_type` (`type`),
  KEY `idx_volume_operations_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping routines for database 'migratekit_oma'
--

--
-- Final view structure for view `active_jobs`
--

/*!50001 DROP VIEW IF EXISTS `active_jobs`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`oma_user`@`localhost` SQL SECURITY DEFINER */
/*!50001 VIEW `active_jobs` AS select `j`.`id` AS `id`,`j`.`job_type` AS `job_type`,`j`.`operation` AS `operation`,`j`.`status` AS `status`,`j`.`parent_job_id` AS `parent_job_id`,`j`.`started_at` AS `started_at`,timestampdiff(SECOND,`j`.`started_at`,current_timestamp()) AS `running_seconds`,(select count(0) from `job_execution_log` where `job_execution_log`.`job_id` = `j`.`id` and `job_execution_log`.`log_level` = 'ERROR') AS `error_count` from `job_tracking` `j` where `j`.`status` in ('pending','running') order by `j`.`started_at` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `active_schedules`
--

/*!50001 DROP VIEW IF EXISTS `active_schedules`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`oma_user`@`localhost` SQL SECURITY DEFINER */
/*!50001 VIEW `active_schedules` AS select `s`.`id` AS `id`,`s`.`name` AS `name`,`s`.`description` AS `description`,`s`.`cron_expression` AS `cron_expression`,`s`.`schedule_type` AS `schedule_type`,`s`.`enabled` AS `enabled`,`s`.`max_concurrent_jobs` AS `max_concurrent_jobs`,count(distinct `g`.`id`) AS `group_count`,count(distinct `m`.`vm_context_id`) AS `vm_count`,case when `s`.`schedule_type` = 'cron' and `s`.`enabled` = 1 then current_timestamp() + interval 1 hour else NULL end AS `next_execution`,(select max(`se`.`started_at`) from `schedule_executions` `se` where `se`.`schedule_id` = `s`.`id`) AS `last_execution` from ((`replication_schedules` `s` left join `vm_machine_groups` `g` on(`s`.`id` = `g`.`schedule_id`)) left join `vm_group_memberships` `m` on(`g`.`id` = `m`.`group_id` and `m`.`enabled` = 1)) where `s`.`enabled` = 1 group by `s`.`id`,`s`.`name`,`s`.`description`,`s`.`cron_expression`,`s`.`schedule_type`,`s`.`enabled`,`s`.`max_concurrent_jobs` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `job_tracking_hierarchy`
--

/*!50001 DROP VIEW IF EXISTS `job_tracking_hierarchy`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`oma_user`@`localhost` SQL SECURITY DEFINER */
/*!50001 VIEW `job_tracking_hierarchy` AS select `j`.`id` AS `id`,`j`.`job_type` AS `job_type`,`j`.`operation` AS `operation`,`j`.`status` AS `status`,`j`.`parent_job_id` AS `parent_job_id`,`j`.`started_at` AS `started_at`,`j`.`completed_at` AS `completed_at`,case when `j`.`parent_job_id` is null then 'root' else 'child' end AS `job_level`,(select count(0) from `job_tracking` where `job_tracking`.`parent_job_id` = `j`.`id`) AS `child_count`,case when `j`.`completed_at` is not null then timestampdiff(SECOND,`j`.`started_at`,`j`.`completed_at`) else NULL end AS `duration_seconds` from `job_tracking` `j` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `schedule_execution_summary`
--

/*!50001 DROP VIEW IF EXISTS `schedule_execution_summary`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`oma_user`@`localhost` SQL SECURITY DEFINER */
/*!50001 VIEW `schedule_execution_summary` AS select `se`.`id` AS `id`,`se`.`schedule_id` AS `schedule_id`,`rs`.`name` AS `schedule_name`,`se`.`status` AS `status`,`se`.`scheduled_at` AS `scheduled_at`,`se`.`started_at` AS `started_at`,`se`.`completed_at` AS `completed_at`,`se`.`execution_duration_seconds` AS `execution_duration_seconds`,`se`.`vms_eligible` AS `vms_eligible`,`se`.`jobs_created` AS `jobs_created`,`se`.`jobs_completed` AS `jobs_completed`,`se`.`jobs_failed` AS `jobs_failed`,`se`.`jobs_skipped` AS `jobs_skipped`,case when `se`.`jobs_created` > 0 then round(`se`.`jobs_completed` / `se`.`jobs_created` * 100,2) else 0 end AS `success_rate_percent`,`se`.`error_message` AS `error_message`,`se`.`triggered_by` AS `triggered_by` from (`schedule_executions` `se` join `replication_schedules` `rs` on(`se`.`schedule_id` = `rs`.`id`)) order by `se`.`scheduled_at` desc */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;

--
-- Final view structure for view `vm_schedule_status`
--

/*!50001 DROP VIEW IF EXISTS `vm_schedule_status`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb3 */;
/*!50001 SET character_set_results     = utf8mb3 */;
/*!50001 SET collation_connection      = utf8mb3_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`oma_user`@`localhost` SQL SECURITY DEFINER */
/*!50001 VIEW `vm_schedule_status` AS select `vrc`.`context_id` AS `context_id`,`vrc`.`vm_name` AS `vm_name`,`vrc`.`current_status` AS `vm_status`,`vrc`.`next_scheduled_at` AS `next_scheduled_at`,`vrc`.`scheduler_enabled` AS `scheduler_enabled`,`vmg`.`name` AS `group_name`,`vmg`.`priority` AS `group_priority`,`vgm`.`priority` AS `vm_priority`,`vgm`.`enabled` AS `membership_enabled`,`rs`.`name` AS `schedule_name`,`rs`.`cron_expression` AS `cron_expression`,`rs`.`enabled` AS `schedule_enabled`,case when `rj`.`id` is not null then 1 else 0 end AS `has_active_job`,`rj`.`id` AS `active_job_id`,`rj`.`status` AS `job_status`,`rj`.`progress_percent` AS `progress_percent` from ((((`vm_replication_contexts` `vrc` left join `vm_group_memberships` `vgm` on(`vrc`.`context_id` = `vgm`.`vm_context_id`)) left join `vm_machine_groups` `vmg` on(`vgm`.`group_id` = `vmg`.`id`)) left join `replication_schedules` `rs` on(`vmg`.`schedule_id` = `rs`.`id` or `vgm`.`schedule_override_id` = `rs`.`id`)) left join `replication_jobs` `rj` on(`vrc`.`current_job_id` = `rj`.`id` and `rj`.`status` in ('replicating','provisioning'))) where `vrc`.`scheduler_enabled` = 1 */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-10-04 12:26:58
