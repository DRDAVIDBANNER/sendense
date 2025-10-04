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
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_vm_network_type` (`vm_id`,`source_network_name`,`is_test_network`),
  KEY `idx_vm_id` (`vm_id`),
  KEY `idx_is_test` (`is_test_network`)
) ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `network_mappings`
--

LOCK TABLES `network_mappings` WRITE;
/*!40000 ALTER TABLE `network_mappings` DISABLE KEYS */;
INSERT INTO `network_mappings` VALUES
(1,'4205a841-0265-f4bd-39a6-39fd92196f53','VM Network','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',1,'2025-08-18 14:25:52','2025-08-18 16:21:32'),
(2,'4205a841-0265-f4bd-39a6-39fd92196f53','Unknown','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',0,'2025-08-19 06:11:46','2025-08-19 06:11:46'),
(6,'4205784a-098a-40f1-1f1e-a5cd2597fd59','VM Network','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',1,'2025-08-19 14:59:16','2025-08-19 14:59:16'),
(7,'420570c7-f61f-a930-77c5-1e876786cb3c','Unknown','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',0,'2025-08-22 07:42:06','2025-08-22 07:42:06'),
(8,'420570c7-f61f-a930-77c5-1e876786cb3c','VM Network','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',1,'2025-08-22 07:48:58','2025-08-22 07:48:58'),
(9,'42056031-ac68-9c35-f13b-12125fcab603','VM Network','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',0,'2025-09-03 17:26:16','2025-09-03 17:40:06'),
(10,'4205eba5-83e9-f0e2-3823-b17e3167a67b','VM Network','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',0,'2025-09-03 17:41:30','2025-09-03 17:41:44'),
(11,'420570c7-f61f-a930-77c5-1e876786cb3c','VLAN 253 - QUADRIS_CLOUD-DMZ','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',0,'2025-09-04 14:52:18','2025-09-04 14:52:18'),
(12,'4205430b-b2ec-35af-0402-4a4c4a49cff6','VLAN 501 - UKFAST-SERVERS','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',0,'2025-09-07 14:07:01','2025-09-07 14:07:01'),
(13,'pgtest1','pgtest1-network','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',0,'2025-09-18 09:02:41','2025-09-18 09:02:41'),
(14,'pgtest2','pgtest2-network','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',0,'2025-09-18 09:02:52','2025-09-18 09:02:52'),
(17,'pgtest1','pgtest1-network','d9e89f6f-b84c-490c-b84a-af576fe6d38c','OSSEA-L2-TEST',1,'2025-09-21 07:09:13','2025-09-21 07:09:13'),
(18,'QCDev-Jump05','QCDev-Jump05-network','802c2d41-9152-47b3-885e-a7e0a924eb6a','OSSEA-L2',0,'2025-09-24 04:35:23','2025-09-24 04:35:23');
/*!40000 ALTER TABLE `network_mappings` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-09-24  5:47:15
