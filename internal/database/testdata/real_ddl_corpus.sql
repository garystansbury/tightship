-- A false-positive corpus for the rollback lint: CREATE TABLE statements taken from a real
-- long-lived schema, with every identifier and string literal replaced.
--
-- What is preserved is the only thing the lint reads: DDL grammar. Column types with commas
-- inside them, NOT NULL and DEFAULT clauses, index and constraint declarations, engine and
-- charset clauses, and the parenthesis nesting the clause splitter has to handle. Regex over
-- SQL trips on real-world shapes far more than on invented ones, and this is the cheapest
-- evidence available that it does not.
--
-- Every statement is a CREATE TABLE, which is always additive, so ANY finding is a false
-- positive. The names carry no information: this repository is public, and the schema they
-- came from is not ours to publish.

CREATE TABLE `n001` (
  `n002` varchar(50) NOT NULL,
  `n003` char(36) NOT NULL,
  `n004` varchar(500) DEFAULT NULL,
  `n005` varchar(100) DEFAULT NULL,
  `n006` varchar(255) DEFAULT NULL,
  `n007` varchar(255) DEFAULT NULL,
  `n008` datetime NOT NULL COMMENT 'v1',
  `n009` datetime DEFAULT NULL COMMENT 'v2',
  PRIMARY KEY (`n003`),
  KEY `n010` (`n002`),
  KEY `n011` (`n009`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='v3';
CREATE TABLE `n012` (
  `n003` char(36) NOT NULL,
  `n002` varchar(50) DEFAULT NULL,
  `n013` varchar(20) DEFAULT NULL COMMENT 'v4',
  `n014` varchar(20) DEFAULT NULL COMMENT 'v5',
  `n015` datetime(3) DEFAULT NULL,
  `n016` varchar(255) DEFAULT NULL,
  `n017` varchar(255) DEFAULT NULL,
  `n018` varchar(500) DEFAULT NULL,
  `n019` longtext DEFAULT NULL,
  `n020` enum('v6','v7') DEFAULT NULL,
  `n021` varchar(255) DEFAULT NULL,
  `n022` varchar(100) DEFAULT NULL,
  `n023` varchar(50) DEFAULT 'v8',
  `n024` varchar(50) DEFAULT 'v9',
  `n025` varchar(32) DEFAULT NULL COMMENT 'v10',
  `n026` date DEFAULT NULL,
  PRIMARY KEY (`n003`),
  KEY `n010` (`n002`),
  KEY `n027` (`n013`),
  KEY `n028` (`n014`),
  KEY `n029` (`n022`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n030` (
  `n031` varchar(30) NOT NULL,
  `n032` varchar(50) NOT NULL,
  `n033` longtext NOT NULL,
  `n034` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`n032`,`n031`),
  KEY `n035` (`n034`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n036` (
  `n037` varchar(30) NOT NULL,
  `n038` int(11) NOT NULL,
  PRIMARY KEY (`n037`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n039` (
  `n040` varchar(128) NOT NULL,
  `n041` varchar(255) NOT NULL,
  `n042` varchar(45) NOT NULL DEFAULT 'v11',
  `n043` varchar(32) NOT NULL,
  `n044` varchar(255) DEFAULT NULL,
  `n045` varchar(255) DEFAULT NULL,
  `n046` datetime NOT NULL DEFAULT current_timestamp(),
  `n047` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n040`),
  KEY `n048` (`n047`),
  KEY `n049` (`n041`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n050` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n052` datetime NOT NULL,
  `n053` varchar(32) NOT NULL,
  `n054` varchar(64) NOT NULL DEFAULT 'v12',
  `n055` varchar(190) DEFAULT NULL,
  `n056` varchar(320) DEFAULT NULL,
  `n042` varchar(64) DEFAULT NULL,
  `n057` varchar(128) DEFAULT NULL,
  `n058` varchar(255) DEFAULT NULL,
  `n059` text DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n060` (`n053`,`n052`),
  KEY `n061` (`n052`),
  KEY `n062` (`n057`)
) ENGINE=InnoDB AUTO_INCREMENT=302840 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n063` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n064` char(64) NOT NULL,
  `n065` varchar(255) DEFAULT NULL,
  `n066` enum('v13','v14') NOT NULL DEFAULT 'v15',
  `n067` enum('v16','v17','v18','v19','v20','v21') NOT NULL DEFAULT 'v22',
  `n068` datetime NOT NULL DEFAULT current_timestamp(),
  `n069` varchar(64) DEFAULT NULL,
  `n070` datetime NOT NULL,
  `n071` datetime DEFAULT NULL,
  `n072` varchar(128) DEFAULT NULL,
  `n073` datetime DEFAULT NULL,
  `n074` datetime DEFAULT NULL,
  `n075` varchar(64) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n076` (`n064`),
  KEY `n077` (`n067`,`n070`),
  KEY `n078` (`n068`)
) ENGINE=InnoDB AUTO_INCREMENT=43 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n079` (
  `n080` varchar(64) NOT NULL,
  `n081` varchar(255) NOT NULL,
  `n082` varchar(255) DEFAULT NULL,
  `n083` varchar(128) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n080`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n085` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n086` datetime NOT NULL DEFAULT current_timestamp(),
  `n087` varchar(48) NOT NULL,
  `n088` varchar(8) NOT NULL DEFAULT 'v23',
  `n067` varchar(16) NOT NULL DEFAULT 'v24',
  `n089` int(11) DEFAULT NULL,
  `n090` int(11) DEFAULT NULL,
  `n091` varchar(64) DEFAULT NULL,
  `n092` text DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n093` (`n087`,`n086`),
  KEY `n094` (`n088`,`n086`)
) ENGINE=InnoDB AUTO_INCREMENT=921134 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n095` (
  `n096` varchar(64) NOT NULL,
  `n097` varchar(64) NOT NULL,
  `n098` varchar(128) NOT NULL,
  `n099` varchar(255) DEFAULT NULL,
  `n100` varchar(255) DEFAULT NULL,
  `n101` varchar(255) DEFAULT NULL,
  `n008` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n096`,`n097`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n102` (
  `n097` varchar(64) NOT NULL,
  `n098` varchar(128) NOT NULL,
  `n099` varchar(255) NOT NULL DEFAULT 'v25',
  `n100` varchar(255) NOT NULL DEFAULT 'v26',
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n097`),
  KEY `n103` (`n098`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n104` (
  `n105` varchar(50) NOT NULL,
  `n106` datetime NOT NULL,
  `n107` datetime NOT NULL,
  `n108` varchar(255) DEFAULT NULL,
  `n109` varchar(16) DEFAULT NULL,
  `n110` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`n105`),
  KEY `n111` (`n109`),
  KEY `n112` (`n106`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n113` (
  `n114` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n097` varchar(64) NOT NULL,
  `n115` varchar(32) NOT NULL,
  `n088` enum('v27','v28') NOT NULL DEFAULT 'v29',
  `n116` datetime DEFAULT current_timestamp(),
  `n117` varchar(128) NOT NULL,
  `n118` date DEFAULT NULL,
  `n119` datetime DEFAULT NULL,
  `n120` varchar(128) DEFAULT NULL,
  `n121` varchar(32) DEFAULT NULL,
  `n053` varchar(32) NOT NULL,
  `n122` tinyint(1) GENERATED ALWAYS AS (if(`n119` is null,1,NULL)) STORED,
  PRIMARY KEY (`n114`),
  UNIQUE KEY `n123` (`n097`,`n122`),
  KEY `n010` (`n097`,`n119`),
  KEY `n124` (`n115`,`n119`),
  KEY `n125` (`n088`,`n119`)
) ENGINE=InnoDB AUTO_INCREMENT=117788 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n126` (
  `n127` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n097` varchar(64) NOT NULL,
  `n128` varchar(64) NOT NULL,
  `n129` datetime NOT NULL DEFAULT current_timestamp(),
  `n130` varchar(128) NOT NULL,
  `n131` datetime DEFAULT NULL,
  `n132` varchar(128) DEFAULT NULL,
  `n053` varchar(32) NOT NULL,
  `n122` tinyint(1) GENERATED ALWAYS AS (if(`n131` is null,1,NULL)) STORED,
  PRIMARY KEY (`n127`),
  UNIQUE KEY `n123` (`n097`,`n122`),
  KEY `n133` (`n128`,`n131`),
  KEY `n010` (`n097`,`n131`)
) ENGINE=InnoDB AUTO_INCREMENT=27233 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n134` (
  `n135` bigint(20) NOT NULL AUTO_INCREMENT,
  `n136` datetime(3) NOT NULL DEFAULT current_timestamp(3),
  `n105` varchar(50) NOT NULL,
  `n098` char(36) DEFAULT NULL,
  `n057` enum('v30','v31','v32','v33','v34','v35','v36','v37','v38','v39','v40','v41','v42','v43','v44') NOT NULL,
  `n137` varchar(64) DEFAULT NULL,
  `n138` enum('v45','v46','v47') NOT NULL DEFAULT 'v48',
  `n043` enum('v49','v50','v51','v52') NOT NULL,
  `n139` enum('v53','v54','v55') NOT NULL DEFAULT 'v56',
  `n140` varchar(255) NOT NULL,
  `n056` varchar(255) DEFAULT NULL,
  `n141` varchar(45) DEFAULT NULL,
  `n142` enum('v57','v58','v59') DEFAULT NULL,
  `n143` char(2) DEFAULT NULL,
  `n144` varchar(3) DEFAULT NULL,
  `n145` varchar(64) DEFAULT NULL,
  `n146` varchar(64) DEFAULT NULL,
  `n147` varchar(16) DEFAULT NULL,
  `n148` char(36) DEFAULT NULL,
  `n149` varchar(500) DEFAULT NULL,
  `n150` varchar(500) DEFAULT NULL,
  `n151` varchar(255) DEFAULT NULL,
  `n152` varchar(255) DEFAULT NULL,
  `n153` varchar(100) DEFAULT NULL,
  `n154` varchar(100) DEFAULT NULL,
  `n155` varchar(50) DEFAULT NULL,
  `n156` varchar(50) DEFAULT NULL,
  `n157` int(11) DEFAULT NULL,
  `n059` text DEFAULT NULL,
  PRIMARY KEY (`n135`),
  KEY `n158` (`n105`,`n136`),
  KEY `n159` (`n136`),
  KEY `n160` (`n148`),
  KEY `n062` (`n057`),
  KEY `n161` (`n140`),
  KEY `n103` (`n098`)
) ENGINE=InnoDB AUTO_INCREMENT=126911 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n162` (
  `n097` varchar(64) NOT NULL,
  `n163` datetime NOT NULL DEFAULT current_timestamp(),
  `n164` varchar(128) NOT NULL,
  `n053` varchar(32) NOT NULL,
  `n165` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n097`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n166` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n167` varchar(16) NOT NULL,
  `n168` varchar(18) NOT NULL,
  `n169` varchar(64) NOT NULL,
  `n170` smallint(6) NOT NULL,
  `n171` tinyint(1) DEFAULT 1,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n172` (`n168`)
) ENGINE=InnoDB AUTO_INCREMENT=285 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n173` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n174` char(36) NOT NULL,
  `n053` enum('v60','v61') NOT NULL,
  `n175` varchar(255) DEFAULT NULL,
  `n091` varchar(64) DEFAULT NULL,
  `n067` enum('v62','v63','v64','v65','v66','v67') NOT NULL DEFAULT 'v68',
  `n068` datetime DEFAULT NULL,
  `n176` datetime DEFAULT NULL,
  `n177` datetime DEFAULT NULL,
  `n090` int(10) unsigned DEFAULT NULL,
  `n089` int(11) DEFAULT NULL,
  `n092` varchar(500) DEFAULT NULL,
  `n178` mediumtext DEFAULT NULL,
  `n179` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n180` (`n174`),
  KEY `n077` (`n067`),
  KEY `n181` (`n179`)
) ENGINE=InnoDB AUTO_INCREMENT=848 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n182` (
  `n183` char(32) NOT NULL,
  `n184` varchar(255) NOT NULL COMMENT 'v69',
  `n185` varchar(255) NOT NULL COMMENT 'v70',
  `n186` datetime NOT NULL COMMENT 'v71',
  `n074` datetime NOT NULL DEFAULT current_timestamp(),
  `n187` varchar(64) DEFAULT NULL COMMENT 'v72',
  PRIMARY KEY (`n183`),
  KEY `n188` (`n186`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='v73';
CREATE TABLE `n189` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n086` int(10) unsigned NOT NULL,
  `n190` varchar(48) NOT NULL,
  `n191` varchar(24) NOT NULL,
  `n192` varchar(96) NOT NULL,
  `n067` enum('v74','v75','v76') NOT NULL,
  `n193` varchar(512) DEFAULT NULL,
  `n194` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n195` (`n086`),
  KEY `n196` (`n190`,`n086`),
  KEY `n197` (`n067`,`n086`),
  KEY `n198` (`n191`,`n086`)
) ENGINE=InnoDB AUTO_INCREMENT=3659051 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n199` (
  `n200` varchar(32) NOT NULL,
  `n201` mediumtext NOT NULL,
  `n054` varchar(64) DEFAULT NULL,
  `n052` int(10) unsigned NOT NULL,
  `n084` timestamp NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n200`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n202` (
  `n203` varchar(64) NOT NULL,
  `n204` varchar(255) NOT NULL,
  `n083` varchar(255) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n203`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n205` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n206` enum('v77','v78','v79') NOT NULL,
  `n207` varchar(64) NOT NULL,
  `n065` varchar(255) NOT NULL,
  `n208` varchar(255) NOT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n073` datetime DEFAULT NULL,
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  `n209` datetime DEFAULT NULL,
  `n210` int(10) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n211` (`n206`,`n207`),
  KEY `n212` (`n171`,`n073`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n213` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n174` char(36) NOT NULL,
  `n053` enum('v80','v81') NOT NULL,
  `n175` varchar(255) DEFAULT NULL,
  `n091` varchar(64) DEFAULT NULL,
  `n067` enum('v82','v83','v84','v85','v86','v87') NOT NULL DEFAULT 'v88',
  `n214` tinyint(4) NOT NULL DEFAULT 0,
  `n068` datetime DEFAULT NULL,
  `n176` datetime DEFAULT NULL,
  `n177` datetime DEFAULT NULL,
  `n090` int(10) unsigned DEFAULT NULL,
  `n215` int(10) unsigned DEFAULT NULL,
  `n216` int(10) unsigned DEFAULT NULL,
  `n217` int(10) unsigned DEFAULT NULL,
  `n218` int(10) unsigned DEFAULT NULL,
  `n219` int(10) unsigned DEFAULT NULL,
  `n220` int(10) unsigned DEFAULT NULL,
  `n221` int(10) unsigned DEFAULT NULL,
  `n222` int(10) unsigned DEFAULT NULL,
  `n223` tinyint(4) NOT NULL DEFAULT 0,
  `n092` varchar(500) DEFAULT NULL,
  `n178` mediumtext DEFAULT NULL,
  `n179` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n180` (`n174`),
  KEY `n077` (`n067`),
  KEY `n181` (`n179`),
  KEY `n224` (`n053`)
) ENGINE=InnoDB AUTO_INCREMENT=347 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n225` (
  `n051` tinyint(3) unsigned NOT NULL DEFAULT 1,
  `n226` tinyint(4) NOT NULL DEFAULT 0,
  `n227` time NOT NULL DEFAULT 'v89',
  `n228` tinyint(4) NOT NULL DEFAULT 1,
  `n229` int(11) NOT NULL DEFAULT 10,
  `n230` varchar(255) NOT NULL DEFAULT 'v90',
  `n231` date DEFAULT NULL,
  `n083` varchar(255) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n051`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n232` (
  `n115` varchar(64) NOT NULL,
  `n041` varchar(255) NOT NULL,
  `n065` varchar(64) NOT NULL,
  `n053` varchar(32) NOT NULL DEFAULT 'v91',
  `n233` varchar(64) NOT NULL,
  `n234` datetime NOT NULL DEFAULT current_timestamp(),
  `n165` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n115`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n235` (
  `n236` varchar(24) NOT NULL,
  `n157` bigint(20) NOT NULL,
  `n237` datetime NOT NULL DEFAULT current_timestamp(),
  `n059` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n236`,`n157`),
  KEY `n238` (`n237`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n239` (
  `n240` int(10) unsigned NOT NULL COMMENT 'v92',
  `n241` varchar(128) DEFAULT NULL COMMENT 'v93',
  `n242` enum('v94','v95','v96','v97','v98','v99','v100') NOT NULL DEFAULT 'v101',
  `n243` varchar(128) DEFAULT NULL,
  `n046` datetime NOT NULL COMMENT 'v102',
  `n047` datetime NOT NULL COMMENT 'v103',
  PRIMARY KEY (`n240`),
  KEY `n244` (`n242`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='v104';
CREATE TABLE `n245` (
  `n240` int(10) unsigned NOT NULL,
  `n246` varchar(128) NOT NULL DEFAULT 'v105',
  `n247` varchar(48) NOT NULL DEFAULT 'v106',
  `n248` varchar(48) NOT NULL DEFAULT 'v107',
  `n249` text DEFAULT NULL,
  `n250` datetime NOT NULL,
  PRIMARY KEY (`n240`),
  KEY `n251` (`n250`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n252` (
  `n253` datetime NOT NULL,
  `n066` varchar(64) NOT NULL,
  `n254` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`n253`,`n066`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n255` (
  `n051` bigint(20) NOT NULL AUTO_INCREMENT,
  `n115` varchar(64) NOT NULL,
  `n256` char(64) NOT NULL,
  `n257` varbinary(32) NOT NULL,
  `n258` char(64) DEFAULT NULL,
  `n259` varchar(255) DEFAULT NULL,
  `n260` char(6) DEFAULT NULL,
  `n192` varchar(64) DEFAULT NULL,
  `n067` enum('v108','v109','v110','v111') NOT NULL DEFAULT 'v112',
  `n261` varchar(64) NOT NULL,
  `n262` datetime NOT NULL DEFAULT current_timestamp(),
  `n263` datetime DEFAULT NULL,
  `n264` varchar(64) DEFAULT NULL,
  `n265` int(11) NOT NULL DEFAULT 0,
  `n266` datetime DEFAULT NULL,
  `n267` datetime DEFAULT NULL,
  `n268` varchar(128) DEFAULT NULL,
  `n269` datetime DEFAULT NULL,
  `n270` datetime DEFAULT NULL,
  `n271` varchar(64) DEFAULT NULL,
  `n272` datetime DEFAULT NULL,
  `n273` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n274` (`n256`),
  UNIQUE KEY `n275` (`n260`),
  KEY `n276` (`n115`,`n067`)
) ENGINE=InnoDB AUTO_INCREMENT=87620 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n277` (
  `n278` bigint(20) NOT NULL,
  `n279` varbinary(255) NOT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n278`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n280` (
  `n281` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n136` datetime NOT NULL DEFAULT current_timestamp(),
  `n097` varchar(64) NOT NULL,
  `n115` varchar(32) NOT NULL,
  `n041` varchar(255) NOT NULL,
  `n282` varchar(8) DEFAULT NULL,
  `n167` varchar(16) DEFAULT NULL,
  `n128` varchar(64) DEFAULT NULL,
  `n283` varchar(128) DEFAULT NULL,
  `n284` varchar(16) DEFAULT NULL,
  `n065` varchar(32) NOT NULL,
  `n059` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n281`),
  KEY `n159` (`n136`),
  KEY `n124` (`n115`),
  KEY `n133` (`n128`),
  KEY `n285` (`n065`)
) ENGINE=InnoDB AUTO_INCREMENT=1145 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n286` (
  `n287` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n136` datetime NOT NULL DEFAULT current_timestamp(),
  `n097` varchar(64) NOT NULL,
  `n115` varchar(32) NOT NULL,
  `n041` varchar(255) NOT NULL,
  `n288` tinyint(1) NOT NULL,
  `n289` varchar(32) NOT NULL,
  `n290` tinyint(1) NOT NULL,
  `n291` varchar(32) NOT NULL,
  `n292` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n287`),
  KEY `n159` (`n136`),
  KEY `n010` (`n097`),
  KEY `n124` (`n115`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n293` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n136` datetime NOT NULL,
  `n041` varchar(255) DEFAULT NULL,
  `n294` varchar(16) NOT NULL,
  `n295` tinyint(1) DEFAULT NULL,
  `n042` varchar(64) DEFAULT NULL,
  `n045` varchar(255) DEFAULT NULL,
  `n296` varchar(64) DEFAULT NULL,
  `n065` varchar(32) DEFAULT NULL COMMENT 'v113',
  `n297` varchar(64) DEFAULT NULL COMMENT 'v114',
  PRIMARY KEY (`n051`),
  KEY `n298` (`n136`),
  KEY `n299` (`n041`),
  KEY `n300` (`n294`,`n136`)
) ENGINE=InnoDB AUTO_INCREMENT=2129 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n301` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n097` varchar(64) NOT NULL COMMENT 'v115',
  `n302` varchar(64) NOT NULL COMMENT 'v116',
  `n303` int(10) unsigned DEFAULT NULL COMMENT 'v117',
  `n304` datetime DEFAULT NULL COMMENT 'v118',
  `n305` varchar(128) NOT NULL,
  `n098` varchar(64) DEFAULT NULL COMMENT 'v119',
  `n101` varchar(500) DEFAULT NULL COMMENT 'v120',
  `n046` datetime NOT NULL COMMENT 'v121',
  `n047` datetime NOT NULL COMMENT 'v122',
  `n306` int(10) unsigned NOT NULL DEFAULT 1,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n307` (`n097`,`n302`),
  KEY `n112` (`n046`),
  KEY `n308` (`n304`),
  KEY `n309` (`n097`,`n046`)
) ENGINE=InnoDB AUTO_INCREMENT=7283876 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n310` (
  `n051` bigint(20) NOT NULL AUTO_INCREMENT,
  `n115` varchar(64) NOT NULL,
  `n311` char(64) NOT NULL,
  `n312` varchar(128) DEFAULT NULL,
  `n262` datetime NOT NULL DEFAULT current_timestamp(),
  `n073` datetime NOT NULL,
  `n263` datetime DEFAULT NULL,
  `n313` varchar(64) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n314` (`n311`),
  KEY `n315` (`n115`,`n263`,`n073`)
) ENGINE=InnoDB AUTO_INCREMENT=2227073 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n316` (
  `n097` varchar(64) NOT NULL COMMENT 'v123',
  `n098` varchar(64) DEFAULT NULL COMMENT 'v124',
  `n302` varchar(64) NOT NULL DEFAULT 'v125' COMMENT 'v126',
  `n304` datetime DEFAULT NULL,
  `n305` varchar(128) NOT NULL COMMENT 'v127',
  `n046` datetime NOT NULL COMMENT 'v128',
  `n047` datetime NOT NULL COMMENT 'v129',
  `n306` int(10) unsigned NOT NULL DEFAULT 1,
  `n317` tinyint(1) NOT NULL DEFAULT 0 COMMENT 'v130',
  PRIMARY KEY (`n097`),
  KEY `n048` (`n047`),
  KEY `n318` (`n317`,`n047`),
  KEY `n308` (`n304`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n319` (
  `n320` date NOT NULL,
  `n167` varchar(16) NOT NULL,
  `n282` varchar(16) NOT NULL,
  `n321` int(11) NOT NULL DEFAULT 0,
  `n322` int(11) NOT NULL DEFAULT 0,
  `n323` int(11) NOT NULL DEFAULT 0,
  `n324` int(11) NOT NULL DEFAULT 0,
  `n325` datetime NOT NULL,
  PRIMARY KEY (`n320`,`n167`,`n282`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n326` (
  `n282` varchar(16) NOT NULL,
  `n327` enum('v131','v132') NOT NULL,
  `n328` enum('v133','v134','v135','v136') NOT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `n329` enum('v137','v138') DEFAULT NULL,
  `n330` tinyint(4) DEFAULT NULL,
  `n331` int(11) DEFAULT NULL,
  PRIMARY KEY (`n282`),
  KEY `n332` (`n331`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n333` (
  `n051` bigint(20) NOT NULL AUTO_INCREMENT,
  `n136` datetime NOT NULL DEFAULT current_timestamp(),
  `n115` varchar(64) DEFAULT NULL,
  `n041` varchar(255) DEFAULT NULL,
  `n282` varchar(16) DEFAULT NULL,
  `n138` varchar(32) NOT NULL,
  `n334` varchar(32) DEFAULT NULL,
  `n335` varchar(128) DEFAULT NULL,
  `n336` varchar(32) NOT NULL DEFAULT 'v139',
  `n337` tinyint(1) NOT NULL DEFAULT 0,
  `n338` varchar(64) DEFAULT NULL COMMENT 'v140',
  `n339` bigint(20) DEFAULT NULL COMMENT 'v141',
  `n340` varchar(16) DEFAULT NULL COMMENT 'v142',
  `n341` varchar(128) DEFAULT NULL COMMENT 'v143',
  `n342` tinyint(1) NOT NULL DEFAULT 0,
  `n295` tinyint(1) DEFAULT NULL,
  `n042` varchar(64) DEFAULT NULL,
  `n045` varchar(255) DEFAULT NULL,
  `n343` varchar(255) DEFAULT NULL,
  `n142` varchar(16) DEFAULT NULL,
  `n143` varchar(2) DEFAULT NULL,
  `n344` varchar(64) DEFAULT NULL,
  `n345` varchar(64) DEFAULT NULL,
  `n146` varchar(96) DEFAULT NULL,
  `n147` varchar(16) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n346` (`n115`,`n136`),
  KEY `n159` (`n136`),
  KEY `n347` (`n336`,`n136`),
  KEY `n348` (`n138`,`n136`),
  KEY `n349` (`n142`,`n136`),
  KEY `n350` (`n143`,`n136`),
  KEY `n351` (`n338`,`n136`),
  KEY `n352` (`n339`),
  KEY `n353` (`n138`,`n136`,`n338`,`n342`,`n041`,`n335`)
) ENGINE=InnoDB AUTO_INCREMENT=2881979 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n354` (
  `n051` bigint(20) NOT NULL AUTO_INCREMENT,
  `n115` varchar(64) NOT NULL,
  `n355` text NOT NULL,
  `n356` char(64) NOT NULL,
  `n357` char(64) NOT NULL,
  `n358` smallint(6) NOT NULL,
  `n067` enum('v144','v145','v146','v147','v148') NOT NULL DEFAULT 'v149',
  `n359` varchar(128) DEFAULT NULL,
  `n360` int(11) NOT NULL DEFAULT 0,
  `n361` varchar(64) DEFAULT NULL,
  `n362` varchar(64) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n073` datetime NOT NULL,
  `n071` datetime DEFAULT NULL,
  `n363` datetime DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n364` (`n356`),
  UNIQUE KEY `n365` (`n357`),
  KEY `n366` (`n115`,`n067`,`n073`)
) ENGINE=InnoDB AUTO_INCREMENT=26765 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n367` (
  `n331` int(11) NOT NULL AUTO_INCREMENT,
  `n368` varchar(32) NOT NULL,
  `n369` varchar(64) NOT NULL,
  `n327` enum('v150','v151') NOT NULL DEFAULT 'v152',
  `n328` enum('v153','v154','v155','v156') NOT NULL,
  `n370` tinyint(4) DEFAULT NULL,
  `n371` enum('v157','v158','v159') NOT NULL DEFAULT 'v160',
  `n372` enum('v161','v162','v163') NOT NULL DEFAULT 'v164',
  `n373` enum('v165','v166','v167') NOT NULL DEFAULT 'v168',
  `n374` enum('v169','v170','v171') NOT NULL DEFAULT 'v172',
  `n375` tinyint(3) unsigned NOT NULL DEFAULT 5,
  `n083` varchar(64) NOT NULL DEFAULT 'v173',
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n331`),
  UNIQUE KEY `n368` (`n368`)
) ENGINE=InnoDB AUTO_INCREMENT=18 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n376` (
  `n377` varchar(255) NOT NULL,
  `n378` tinyint(1) NOT NULL,
  `n083` varchar(255) DEFAULT NULL,
  `n084` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n377`),
  KEY `n379` (`n378`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n380` (
  `n115` varchar(64) NOT NULL,
  `n381` varchar(255) NOT NULL,
  `n382` varchar(255) DEFAULT NULL,
  `n383` varchar(512) DEFAULT NULL,
  `n384` int(11) NOT NULL DEFAULT 0,
  `n385` datetime DEFAULT NULL,
  `n083` varchar(64) NOT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n115`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n386` (
  `n051` bigint(20) NOT NULL AUTO_INCREMENT,
  `n115` varchar(64) NOT NULL,
  `n381` varchar(255) NOT NULL,
  `n387` varchar(64) DEFAULT NULL,
  `n388` datetime DEFAULT NULL,
  `n389` datetime NOT NULL DEFAULT current_timestamp(),
  `n390` varchar(64) NOT NULL,
  PRIMARY KEY (`n051`),
  KEY `n391` (`n115`,`n051`)
) ENGINE=InnoDB AUTO_INCREMENT=215 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n392` (
  `n115` varchar(64) NOT NULL,
  `n041` varchar(255) DEFAULT NULL,
  `n393` datetime NOT NULL DEFAULT current_timestamp(),
  `n073` datetime NOT NULL,
  `n394` varchar(64) NOT NULL,
  `n053` varchar(32) NOT NULL DEFAULT 'v174',
  PRIMARY KEY (`n115`),
  KEY `n395` (`n073`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n396` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n136` datetime NOT NULL,
  `n105` varchar(64) NOT NULL,
  `n377` varchar(255) NOT NULL,
  `n397` varchar(16) NOT NULL,
  `n294` varchar(16) NOT NULL,
  `n296` varchar(64) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n398` (`n136`),
  KEY `n399` (`n105`),
  KEY `n400` (`n377`),
  KEY `n401` (`n294`,`n136`)
) ENGINE=InnoDB AUTO_INCREMENT=713 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n402` (
  `n051` bigint(20) NOT NULL AUTO_INCREMENT,
  `n115` varchar(64) NOT NULL,
  `n041` varchar(255) NOT NULL,
  `n065` varchar(64) NOT NULL,
  `n403` varchar(64) NOT NULL,
  `n068` datetime NOT NULL DEFAULT current_timestamp(),
  `n360` int(11) NOT NULL DEFAULT 0,
  `n067` enum('v175','v176','v177') NOT NULL DEFAULT 'v178',
  `n404` datetime DEFAULT NULL,
  `n405` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n077` (`n067`,`n068`)
) ENGINE=InnoDB AUTO_INCREMENT=248 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n406` (
  `n051` bigint(20) NOT NULL AUTO_INCREMENT,
  `n407` char(64) NOT NULL,
  `n115` varchar(64) NOT NULL,
  `n041` varchar(255) NOT NULL,
  `n262` datetime NOT NULL DEFAULT current_timestamp(),
  `n047` datetime NOT NULL DEFAULT current_timestamp(),
  `n073` datetime NOT NULL,
  `n263` datetime DEFAULT NULL,
  `n313` varchar(64) DEFAULT NULL,
  `n042` varchar(64) DEFAULT NULL,
  `n045` varchar(255) DEFAULT NULL,
  `n408` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n409` (`n407`),
  KEY `n315` (`n115`,`n263`,`n073`),
  KEY `n410` (`n263`,`n073`) COMMENT 'v179'
) ENGINE=InnoDB AUTO_INCREMENT=2238815 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n411` (
  `n080` varchar(64) NOT NULL,
  `n081` varchar(255) DEFAULT NULL,
  `n083` varchar(255) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n080`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n412` (
  `n115` varchar(64) NOT NULL,
  `n088` varchar(32) DEFAULT NULL,
  `n381` varchar(255) NOT NULL,
  `n383` varchar(512) NOT NULL,
  `n384` int(11) NOT NULL DEFAULT 0,
  `n385` datetime DEFAULT NULL,
  `n083` varchar(64) NOT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n115`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n413` (
  `n051` bigint(20) NOT NULL AUTO_INCREMENT,
  `n115` varchar(64) NOT NULL,
  `n381` varchar(255) NOT NULL,
  `n387` varchar(64) DEFAULT NULL,
  `n388` datetime DEFAULT NULL,
  `n389` datetime NOT NULL DEFAULT current_timestamp(),
  `n390` varchar(64) NOT NULL,
  PRIMARY KEY (`n051`),
  KEY `n391` (`n115`,`n051`)
) ENGINE=InnoDB AUTO_INCREMENT=164 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n414` (
  `n415` varchar(32) NOT NULL,
  `n105` varchar(32) DEFAULT NULL,
  `n115` varchar(64) DEFAULT NULL,
  `n416` varbinary(6) NOT NULL,
  `n417` varbinary(255) NOT NULL,
  `n067` enum('v180','v181','v182','v183','v184') NOT NULL DEFAULT 'v185',
  `n192` varchar(64) DEFAULT NULL,
  `n418` int(11) DEFAULT NULL,
  `n419` smallint(6) DEFAULT NULL,
  `n420` int(11) NOT NULL DEFAULT 0,
  `n421` datetime DEFAULT NULL,
  `n261` varchar(64) DEFAULT NULL,
  `n262` datetime DEFAULT NULL,
  `n263` datetime DEFAULT NULL,
  `n264` varchar(64) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n415`),
  KEY `n276` (`n115`,`n067`),
  KEY `n010` (`n105`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n422` (
  `n369` varchar(64) NOT NULL,
  `n423` text NOT NULL,
  `n424` date DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `n083` varchar(128) DEFAULT NULL,
  `n165` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n369`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n425` (
  `n157` int(11) NOT NULL AUTO_INCREMENT,
  `n426` varchar(64) NOT NULL,
  `n427` enum('v186','v187') NOT NULL,
  `n428` tinyint(1) DEFAULT 0,
  `n176` datetime DEFAULT current_timestamp(),
  `n429` datetime DEFAULT NULL,
  `n430` datetime DEFAULT NULL,
  `n431` varchar(64) DEFAULT NULL,
  `n208` varchar(64) NOT NULL,
  `n067` enum('v188','v189','v190') DEFAULT 'v191',
  PRIMARY KEY (`n157`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n432` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n157` int(11) NOT NULL,
  `n002` varchar(50) NOT NULL,
  `n017` varchar(255) DEFAULT NULL,
  `n018` varchar(500) DEFAULT NULL,
  `n128` varchar(64) DEFAULT NULL,
  `n167` varchar(16) DEFAULT NULL,
  `n023` varchar(50) DEFAULT NULL,
  `n024` varchar(50) DEFAULT NULL,
  `n026` date DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n433` (`n157`),
  KEY `n010` (`n002`),
  KEY `n111` (`n167`),
  KEY `n133` (`n128`),
  CONSTRAINT `n434` FOREIGN KEY (`n157`) REFERENCES `n425` (`n157`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n435` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n157` int(11) NOT NULL,
  `n128` varchar(64) NOT NULL,
  `n436` varchar(64) NOT NULL,
  `n437` datetime DEFAULT current_timestamp(),
  `n082` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n438` (`n157`,`n128`),
  CONSTRAINT `n439` FOREIGN KEY (`n157`) REFERENCES `n425` (`n157`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n440` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n157` int(11) NOT NULL,
  `n002` varchar(50) NOT NULL,
  `n128` varchar(64) DEFAULT NULL,
  `n167` varchar(16) NOT NULL,
  `n441` enum('v192','v193','v194') DEFAULT 'v195',
  `n442` varchar(64) NOT NULL,
  `n443` datetime DEFAULT current_timestamp(),
  `n444` enum('v196','v197') DEFAULT 'v198',
  `n445` enum('v199','v200','v201') DEFAULT 'v202',
  `n446` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n433` (`n157`),
  KEY `n010` (`n002`),
  KEY `n133` (`n128`),
  KEY `n111` (`n167`),
  CONSTRAINT `n447` FOREIGN KEY (`n157`) REFERENCES `n425` (`n157`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n448` (
  `n087` varchar(48) NOT NULL,
  `n226` tinyint(1) NOT NULL DEFAULT 0,
  `n088` enum('v203','v204','v205','v206') NOT NULL,
  `n449` smallint(5) unsigned DEFAULT NULL,
  `n450` time DEFAULT NULL,
  `n451` tinyint(4) DEFAULT NULL,
  `n452` enum('v207','v208') NOT NULL DEFAULT 'v209',
  `n453` varchar(255) NOT NULL,
  `n454` varchar(512) DEFAULT NULL COMMENT 'v210',
  `n455` int(10) unsigned NOT NULL DEFAULT 3600,
  `n456` tinyint(1) NOT NULL DEFAULT 0,
  `n457` time DEFAULT NULL,
  `n458` datetime DEFAULT NULL,
  `n459` varchar(190) DEFAULT NULL,
  `n460` datetime DEFAULT NULL,
  `n461` datetime DEFAULT NULL,
  `n110` varchar(16) DEFAULT NULL,
  `n462` varchar(64) DEFAULT NULL,
  `n082` varchar(255) DEFAULT NULL,
  `n083` varchar(255) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n087`),
  KEY `n463` (`n226`,`n088`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n464` (
  `n465` varchar(64) NOT NULL,
  `n466` varchar(128) NOT NULL,
  `n467` datetime NOT NULL,
  `n073` datetime NOT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n465`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n468` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n469` enum('v211','v212','v213','v214') NOT NULL,
  `n115` varchar(20) NOT NULL,
  `n470` varchar(255) DEFAULT NULL,
  `n471` varchar(64) DEFAULT NULL,
  `n472` varchar(64) DEFAULT NULL,
  `n473` varchar(64) DEFAULT NULL,
  `n474` varchar(32) DEFAULT NULL,
  `n475` varchar(128) DEFAULT NULL,
  `n476` varchar(128) DEFAULT NULL,
  `n477` varchar(128) DEFAULT NULL,
  `n478` varchar(64) DEFAULT NULL,
  `n479` datetime DEFAULT NULL,
  `n480` tinyint(1) DEFAULT 0,
  `n067` enum('v215','v216','v217','v218') DEFAULT 'v219',
  `n208` varchar(64) DEFAULT NULL,
  `n179` datetime DEFAULT current_timestamp(),
  `n481` datetime DEFAULT NULL,
  `n482` text DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n483` (`n067`,`n479`),
  KEY `n124` (`n115`)
) ENGINE=InnoDB AUTO_INCREMENT=1271 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n484` (
  `n485` int(10) unsigned NOT NULL,
  `n486` int(10) unsigned NOT NULL,
  PRIMARY KEY (`n485`,`n486`),
  KEY `n224` (`n486`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n487` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n471` varchar(64) NOT NULL,
  `n488` varchar(128) NOT NULL,
  `n489` varchar(255) DEFAULT NULL,
  `n490` enum('v220','v221','v222') NOT NULL DEFAULT 'v223',
  `n491` enum('v224','v225','v226') DEFAULT NULL,
  `n492` enum('v227','v228','v229') DEFAULT NULL,
  `n493` varchar(128) DEFAULT NULL,
  `n494` enum('v230','v231') DEFAULT NULL,
  `n495` enum('v232','v233') NOT NULL DEFAULT 'v234',
  `n496` enum('v235','v236') NOT NULL DEFAULT 'v237',
  `n497` text DEFAULT NULL,
  `n498` varchar(32) DEFAULT NULL,
  `n499` varchar(128) DEFAULT NULL,
  `n500` text DEFAULT NULL,
  `n501` varchar(255) DEFAULT NULL,
  `n502` int(11) DEFAULT NULL,
  `n067` enum('v238','v239','v240','v241','v242','v243','v244','v245') NOT NULL DEFAULT 'v246',
  `n503` enum('v247','v248') NOT NULL DEFAULT 'v249',
  `n208` varchar(128) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n504` datetime DEFAULT NULL,
  `n505` datetime DEFAULT NULL,
  `n506` varchar(64) DEFAULT NULL,
  `n082` text DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n507` (`n471`),
  KEY `n077` (`n067`),
  KEY `n508` (`n503`),
  KEY `n509` (`n490`)
) ENGINE=InnoDB AUTO_INCREMENT=190 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n510` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n471` varchar(64) NOT NULL,
  `n511` enum('v250','v251','v252') NOT NULL,
  `n512` varchar(64) DEFAULT NULL,
  `n136` datetime NOT NULL,
  `n513` varchar(191) NOT NULL DEFAULT 'v253',
  `n514` varchar(512) NOT NULL DEFAULT 'v254',
  `n515` bigint(20) unsigned NOT NULL DEFAULT 0,
  `n516` varchar(64) NOT NULL DEFAULT 'v255',
  `n490` enum('v256','v257','v258') DEFAULT NULL,
  `n138` enum('v259','v260','v261') NOT NULL DEFAULT 'v262',
  `n059` text DEFAULT NULL,
  `n517` text DEFAULT NULL,
  `n518` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n519` (`n471`,`n511`,`n136`,`n513`,`n516`,`n514`(150)),
  KEY `n520` (`n471`,`n136`),
  KEY `n521` (`n511`,`n136`)
) ENGINE=InnoDB AUTO_INCREMENT=12346514 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n522` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n369` varchar(64) NOT NULL,
  `n192` varchar(128) NOT NULL,
  `n523` varchar(64) NOT NULL,
  `n054` varchar(255) NOT NULL,
  `n524` int(10) unsigned NOT NULL DEFAULT 22,
  `n525` varchar(191) NOT NULL,
  `n088` enum('v263','v264') NOT NULL DEFAULT 'v265',
  `n526` varchar(191) DEFAULT NULL,
  `n527` text DEFAULT NULL,
  `n528` varchar(8) DEFAULT NULL,
  `n529` varchar(255) DEFAULT NULL,
  `n082` text DEFAULT NULL,
  `n067` enum('v266','v267','v268') NOT NULL DEFAULT 'v269',
  `n530` datetime DEFAULT NULL,
  `n531` enum('v270','v271','v272','v273') NOT NULL DEFAULT 'v274',
  `n532` varchar(255) DEFAULT NULL,
  `n533` datetime DEFAULT NULL,
  `n534` varchar(128) DEFAULT NULL,
  `n535` date DEFAULT NULL,
  `n536` datetime DEFAULT NULL,
  `n537` tinyint(4) DEFAULT NULL,
  `n538` text DEFAULT NULL,
  `n208` varchar(128) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n083` varchar(128) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n539` (`n369`),
  KEY `n540` (`n523`),
  KEY `n077` (`n067`)
) ENGINE=InnoDB AUTO_INCREMENT=38 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n541` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n009` datetime NOT NULL DEFAULT current_timestamp(),
  `n542` varchar(64) DEFAULT NULL,
  `n543` varchar(64) DEFAULT NULL,
  `n053` varchar(64) DEFAULT NULL,
  `n059` text DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n011` (`n009`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n544` (
  `n545` int(10) unsigned NOT NULL,
  `n546` int(10) unsigned NOT NULL,
  PRIMARY KEY (`n545`,`n546`),
  KEY `n547` (`n546`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n548` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n545` int(10) unsigned NOT NULL,
  `n471` varchar(32) NOT NULL,
  `n136` datetime NOT NULL,
  `n138` enum('v275','v276','v277') NOT NULL DEFAULT 'v278',
  `n053` enum('v279','v280') NOT NULL DEFAULT 'v281',
  `n549` varchar(255) DEFAULT NULL,
  `n059` varchar(255) DEFAULT NULL,
  `n517` text DEFAULT NULL,
  `n550` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n551` (`n545`,`n136`,`n053`),
  KEY `n552` (`n545`,`n136`),
  KEY `n553` (`n471`,`n136`),
  KEY `n554` (`n136`)
) ENGINE=InnoDB AUTO_INCREMENT=211700 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n555` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n485` int(10) unsigned DEFAULT NULL,
  `n471` varchar(64) NOT NULL,
  `n369` varchar(128) DEFAULT NULL,
  `n556` enum('v282','v283','v284','v285') NOT NULL DEFAULT 'v286',
  `n453` text NOT NULL,
  `n557` varchar(128) NOT NULL,
  `n226` tinyint(4) NOT NULL DEFAULT 1,
  `n503` enum('v287','v288') NOT NULL DEFAULT 'v289',
  `n558` varchar(255) DEFAULT NULL,
  `n559` tinyint(1) NOT NULL DEFAULT 0,
  `n389` datetime DEFAULT NULL,
  `n390` varchar(128) DEFAULT NULL,
  `n560` datetime DEFAULT NULL,
  `n561` enum('v290','v291','v292') NOT NULL DEFAULT 'v293',
  `n208` varchar(128) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n562` (`n471`,`n453`(191),`n557`),
  KEY `n563` (`n485`),
  KEY `n564` (`n226`),
  KEY `n565` (`n559`)
) ENGINE=InnoDB AUTO_INCREMENT=107 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n566` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n485` int(10) unsigned DEFAULT NULL,
  `n469` enum('v294','v295','v296','v297','v298','v299','v300','v301','v302','v303','v304','v305','v306','v307','v308','v309') NOT NULL,
  `n201` text DEFAULT NULL,
  `n067` enum('v310','v311','v312','v313') NOT NULL DEFAULT 'v314',
  `n479` datetime NOT NULL DEFAULT current_timestamp(),
  `n567` varchar(64) DEFAULT NULL,
  `n568` datetime DEFAULT NULL,
  `n360` int(11) NOT NULL DEFAULT 0,
  `n482` text DEFAULT NULL,
  `n569` text DEFAULT NULL,
  `n208` varchar(128) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n481` datetime DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n570` (`n067`,`n479`),
  KEY `n563` (`n485`)
) ENGINE=InnoDB AUTO_INCREMENT=15938 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n571` (
  `n080` varchar(64) NOT NULL,
  `n081` text DEFAULT NULL,
  `n083` varchar(128) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n080`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n572` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n573` varchar(64) NOT NULL,
  `n574` varchar(64) NOT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n575` (`n573`,`n574`),
  KEY `n576` (`n573`)
) ENGINE=InnoDB AUTO_INCREMENT=60325 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n577` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n369` varchar(64) NOT NULL,
  `n192` varchar(128) DEFAULT NULL,
  `n578` text NOT NULL,
  `n082` text DEFAULT NULL,
  `n208` varchar(128) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n539` (`n369`)
) ENGINE=InnoDB AUTO_INCREMENT=28 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n579` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n296` varchar(64) NOT NULL,
  `n474` varchar(16) NOT NULL,
  `n052` int(10) unsigned NOT NULL,
  `n580` decimal(5,1) DEFAULT NULL,
  `n581` decimal(6,2) DEFAULT NULL,
  `n582` tinyint(3) unsigned DEFAULT NULL,
  `n583` decimal(5,1) DEFAULT NULL,
  `n584` int(10) unsigned DEFAULT NULL,
  `n585` int(10) unsigned DEFAULT NULL,
  `n586` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`n586`)),
  PRIMARY KEY (`n051`),
  KEY `n587` (`n296`,`n052`),
  KEY `n061` (`n052`)
) ENGINE=InnoDB AUTO_INCREMENT=1757351 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n588` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n589` char(64) NOT NULL,
  `n041` varchar(128) NOT NULL,
  `n191` varchar(48) NOT NULL,
  `n590` varchar(64) NOT NULL,
  `n378` varchar(191) NOT NULL,
  `n073` datetime NOT NULL,
  `n074` datetime DEFAULT NULL,
  `n591` varchar(32) DEFAULT NULL,
  `n075` varchar(64) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n592` (`n589`),
  KEY `n593` (`n590`)
) ENGINE=InnoDB AUTO_INCREMENT=40 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n594` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n041` varchar(128) NOT NULL,
  `n191` varchar(48) NOT NULL,
  `n595` enum('v315','v316') NOT NULL,
  `n596` tinyint(4) NOT NULL DEFAULT 0,
  `n059` varchar(255) DEFAULT NULL,
  `n597` varchar(191) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  KEY `n238` (`n179`),
  KEY `n049` (`n041`,`n179`)
) ENGINE=InnoDB AUTO_INCREMENT=3962 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n598` (
  `n041` varchar(128) NOT NULL,
  `n191` varchar(48) NOT NULL,
  `n599` tinyint(4) NOT NULL DEFAULT 0,
  `n600` tinyint(4) NOT NULL DEFAULT 0,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n041`,`n191`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n601` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n041` varchar(128) NOT NULL,
  `n602` text NOT NULL,
  `n603` char(64) NOT NULL,
  `n604` varchar(255) NOT NULL,
  `n605` varchar(64) NOT NULL,
  `n312` varchar(128) DEFAULT NULL,
  `n045` varchar(255) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n606` datetime DEFAULT NULL,
  `n405` varchar(255) DEFAULT NULL,
  `n607` int(11) NOT NULL DEFAULT 0,
  `n263` datetime DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n608` (`n603`),
  KEY `n049` (`n041`),
  KEY `n609` (`n041`,`n263`)
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n610` (
  `n611` tinyint(3) unsigned NOT NULL AUTO_INCREMENT,
  `n612` varchar(32) NOT NULL,
  `n613` varchar(128) DEFAULT NULL,
  PRIMARY KEY (`n611`),
  UNIQUE KEY `n612` (`n612`)
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n614` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n615` varchar(64) NOT NULL,
  `n128` varchar(64) NOT NULL,
  `n616` varchar(64) NOT NULL,
  `n617` datetime DEFAULT current_timestamp(),
  `n171` tinyint(1) DEFAULT 1,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n618` (`n615`,`n128`),
  KEY `n619` (`n615`),
  KEY `n128` (`n128`),
  CONSTRAINT `n620` FOREIGN KEY (`n615`) REFERENCES `n621` (`n615`),
  CONSTRAINT `n622` FOREIGN KEY (`n128`) REFERENCES `n623` (`n128`)
) ENGINE=InnoDB AUTO_INCREMENT=3950 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n624` (
  `n128` varchar(64) NOT NULL,
  `n282` varchar(8) NOT NULL,
  PRIMARY KEY (`n128`,`n282`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n625` (
  `n128` varchar(64) NOT NULL,
  `n626` tinyint(1) NOT NULL DEFAULT 0,
  `n165` varchar(255) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `n083` varchar(128) NOT NULL,
  PRIMARY KEY (`n128`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n623` (
  `n128` varchar(64) NOT NULL,
  `n283` varchar(128) NOT NULL,
  `n167` varchar(16) NOT NULL,
  `n627` enum('v317','v318','v319','v320') DEFAULT 'v321',
  `n628` tinyint(1) NOT NULL DEFAULT 0,
  `n629` datetime DEFAULT NULL,
  `n171` tinyint(1) DEFAULT 1,
  `n179` datetime DEFAULT current_timestamp(),
  `n084` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n128`),
  KEY `n111` (`n167`),
  CONSTRAINT `n630` FOREIGN KEY (`n167`) REFERENCES `n631` (`n167`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n632` (
  `n633` varchar(64) NOT NULL,
  `n597` varchar(255) NOT NULL,
  `n634` varchar(64) DEFAULT NULL,
  `n635` varchar(32) DEFAULT NULL,
  `n636` varchar(64) DEFAULT NULL,
  `n637` varchar(64) DEFAULT NULL,
  `n638` varchar(128) DEFAULT NULL,
  `n639` varchar(64) DEFAULT NULL,
  `n640` varchar(128) DEFAULT NULL,
  `n067` varchar(16) NOT NULL DEFAULT 'v322',
  `n047` datetime NOT NULL,
  PRIMARY KEY (`n633`),
  KEY `n111` (`n637`),
  KEY `n641` (`n639`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n642` (
  `n633` varchar(64) NOT NULL,
  `n643` varchar(64) NOT NULL,
  `n474` enum('v323','v324') NOT NULL,
  `n644` tinyint(1) NOT NULL DEFAULT 0,
  `n067` varchar(16) NOT NULL DEFAULT 'v325',
  `n047` datetime NOT NULL,
  PRIMARY KEY (`n633`,`n643`,`n474`),
  KEY `n553` (`n643`),
  KEY `n244` (`n633`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n645` (
  `n643` varchar(64) NOT NULL,
  `n474` enum('v326','v327') NOT NULL,
  `n041` varchar(255) DEFAULT NULL,
  `n646` varchar(128) DEFAULT NULL,
  `n647` varchar(128) DEFAULT NULL,
  `n282` varchar(16) DEFAULT NULL,
  `n115` varchar(64) DEFAULT NULL,
  `n047` datetime NOT NULL,
  PRIMARY KEY (`n643`),
  KEY `n049` (`n041`),
  KEY `n124` (`n115`),
  KEY `n648` (`n474`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n631` (
  `n167` varchar(16) NOT NULL,
  `n638` varchar(128) NOT NULL,
  `n649` enum('v328','v329','v330','v331') NOT NULL,
  `n650` varchar(64) DEFAULT NULL,
  `n171` tinyint(1) DEFAULT 1,
  `n179` datetime DEFAULT current_timestamp(),
  `n084` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `n637` varchar(64) DEFAULT NULL,
  PRIMARY KEY (`n167`),
  KEY `n651` (`n637`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n652` (
  `n653` varchar(128) NOT NULL,
  `n201` mediumblob NOT NULL,
  `n654` int(10) unsigned NOT NULL,
  PRIMARY KEY (`n653`),
  KEY `n655` (`n654`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n656` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n368` varchar(64) NOT NULL,
  `n369` varchar(128) NOT NULL,
  `n657` text DEFAULT NULL,
  `n658` mediumtext NOT NULL,
  `n659` int(10) unsigned DEFAULT NULL,
  `n067` enum('v332','v333','v334') NOT NULL DEFAULT 'v335',
  `n660` int(10) unsigned NOT NULL DEFAULT 20000,
  `n661` int(10) unsigned NOT NULL DEFAULT 1000,
  `n536` datetime DEFAULT NULL,
  `n662` varchar(191) DEFAULT NULL,
  `n663` int(10) unsigned DEFAULT NULL,
  `n664` int(10) unsigned DEFAULT NULL,
  `n208` varchar(191) NOT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n083` varchar(191) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n665` (`n368`),
  KEY `n077` (`n067`)
) ENGINE=InnoDB AUTO_INCREMENT=55 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n666` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n667` int(10) unsigned NOT NULL,
  `n668` varchar(191) NOT NULL,
  `n669` int(10) unsigned DEFAULT NULL,
  `n165` varchar(255) DEFAULT NULL,
  `n073` datetime DEFAULT NULL,
  `n616` varchar(191) NOT NULL,
  `n617` datetime NOT NULL DEFAULT current_timestamp(),
  `n264` varchar(191) DEFAULT NULL,
  `n263` datetime DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n670` (`n667`,`n668`),
  KEY `n671` (`n668`),
  KEY `n672` (`n667`)
) ENGINE=InnoDB AUTO_INCREMENT=55 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n673` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n667` int(10) unsigned NOT NULL,
  `n674` smallint(5) unsigned NOT NULL,
  `n369` varchar(64) NOT NULL,
  `n675` varchar(48) NOT NULL,
  `n676` varchar(255) DEFAULT NULL,
  `n294` enum('v336','v337') NOT NULL DEFAULT 'v338',
  `n677` varchar(255) DEFAULT NULL,
  `n678` tinyint(4) NOT NULL DEFAULT 0,
  `n192` varchar(128) DEFAULT NULL,
  `n679` enum('v339','v340','v341','v342','v343','v344') NOT NULL DEFAULT 'v345',
  `n680` text DEFAULT NULL,
  `n681` tinyint(4) NOT NULL DEFAULT 0,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n682` (`n667`,`n369`),
  KEY `n683` (`n667`,`n674`)
) ENGINE=InnoDB AUTO_INCREMENT=181 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n684` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n667` int(10) unsigned NOT NULL,
  `n685` int(10) unsigned DEFAULT NULL,
  `n140` varchar(191) NOT NULL,
  `n686` enum('v346','v347','v348') NOT NULL DEFAULT 'v349',
  `n687` text DEFAULT NULL,
  `n176` datetime NOT NULL DEFAULT current_timestamp(),
  `n177` datetime DEFAULT NULL,
  `n688` int(10) unsigned DEFAULT NULL,
  `n689` int(10) unsigned DEFAULT NULL,
  `n690` tinyint(4) NOT NULL DEFAULT 0,
  `n596` tinyint(4) NOT NULL DEFAULT 0,
  `n059` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n691` (`n667`,`n176`),
  KEY `n692` (`n140`,`n176`),
  KEY `n693` (`n177`)
) ENGINE=InnoDB AUTO_INCREMENT=292 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n694` (
  `n115` varchar(32) NOT NULL,
  `n041` varchar(255) NOT NULL,
  `n646` varchar(128) DEFAULT NULL,
  `n695` varchar(128) DEFAULT NULL,
  `n696` varchar(256) DEFAULT NULL,
  `n697` varchar(128) DEFAULT NULL,
  `n698` varchar(512) DEFAULT NULL,
  `n167` varchar(32) DEFAULT NULL,
  `n699` tinyint(1) NOT NULL DEFAULT 0,
  `n084` datetime NOT NULL,
  PRIMARY KEY (`n115`),
  KEY `n049` (`n041`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n700` (
  `n115` varchar(64) NOT NULL,
  `n701` tinyint(4) NOT NULL DEFAULT 0,
  `n387` varchar(255) DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `n165` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n115`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n702` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n115` varchar(64) NOT NULL,
  `n703` varchar(15) NOT NULL,
  `n067` enum('v350','v351','v352') NOT NULL DEFAULT 'v353',
  `n171` tinyint(4) NOT NULL DEFAULT 1,
  `n117` varchar(255) NOT NULL,
  `n116` datetime NOT NULL DEFAULT current_timestamp(),
  `n704` varchar(255) DEFAULT NULL,
  `n705` datetime DEFAULT NULL,
  `n082` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n706` (`n115`,`n703`),
  KEY `n707` (`n171`,`n115`),
  KEY `n708` (`n703`,`n171`)
) ENGINE=InnoDB AUTO_INCREMENT=284 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n709` (
  `n041` varchar(128) NOT NULL,
  `n115` varchar(32) DEFAULT NULL,
  `n710` date NOT NULL,
  `n711` varchar(64) DEFAULT NULL,
  `n712` smallint(6) DEFAULT NULL,
  `n053` varchar(32) NOT NULL DEFAULT 'v354',
  `n084` datetime NOT NULL,
  PRIMARY KEY (`n041`),
  KEY `n124` (`n115`),
  KEY `n713` (`n710`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n321` (
  `n115` varchar(32) NOT NULL,
  `n041` varchar(255) NOT NULL,
  `n646` varchar(128) DEFAULT NULL,
  `n695` varchar(128) DEFAULT NULL,
  `n282` varchar(8) DEFAULT NULL,
  `n698` varchar(512) DEFAULT NULL,
  `n167` varchar(32) DEFAULT NULL,
  `n084` datetime NOT NULL,
  PRIMARY KEY (`n115`),
  KEY `n049` (`n041`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n714` (
  `n115` varchar(32) NOT NULL,
  `n041` varchar(255) DEFAULT NULL,
  `n646` varchar(128) DEFAULT NULL,
  `n695` varchar(128) DEFAULT NULL,
  `n282` varchar(16) DEFAULT NULL,
  `n167` varchar(16) DEFAULT NULL,
  `n698` varchar(512) DEFAULT NULL,
  `n715` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n115`),
  KEY `n049` (`n041`),
  KEY `n716` (`n715`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n717` (
  `n718` int(11) NOT NULL AUTO_INCREMENT,
  `n719` datetime DEFAULT current_timestamp(),
  `n720` enum('v355','v356','v357') NOT NULL,
  `n721` text DEFAULT NULL,
  `n722` varchar(64) NOT NULL,
  PRIMARY KEY (`n718`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n723` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n041` varchar(255) NOT NULL,
  `n724` varchar(64) NOT NULL,
  `n725` datetime NOT NULL DEFAULT current_timestamp(),
  `n073` datetime NOT NULL,
  `n131` datetime DEFAULT NULL,
  `n132` varchar(64) DEFAULT NULL,
  `n360` int(11) NOT NULL DEFAULT 0,
  `n067` enum('v358','v359','v360','v361') NOT NULL DEFAULT 'v362',
  PRIMARY KEY (`n051`),
  KEY `n726` (`n067`,`n073`),
  KEY `n049` (`n041`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n727` (
  `n728` varchar(128) NOT NULL,
  `n729` varchar(64) NOT NULL,
  `n730` varchar(2000) NOT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n728`,`n729`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n731` (
  `n051` int(11) NOT NULL AUTO_INCREMENT,
  `n615` varchar(64) NOT NULL,
  `n611` tinyint(3) unsigned NOT NULL,
  `n167` varchar(16) DEFAULT NULL,
  `n616` varchar(64) NOT NULL,
  `n617` datetime DEFAULT current_timestamp(),
  `n171` tinyint(1) DEFAULT 1,
  `n644` tinyint(1) DEFAULT NULL COMMENT 'v363',
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n732` (`n615`,`n611`,`n167`),
  UNIQUE KEY `n733` (`n611`,`n167`,`n644`),
  KEY `n619` (`n615`),
  KEY `n111` (`n167`),
  KEY `n611` (`n611`),
  CONSTRAINT `n734` FOREIGN KEY (`n615`) REFERENCES `n621` (`n615`),
  CONSTRAINT `n735` FOREIGN KEY (`n611`) REFERENCES `n610` (`n611`),
  CONSTRAINT `n736` FOREIGN KEY (`n167`) REFERENCES `n631` (`n167`)
) ENGINE=InnoDB AUTO_INCREMENT=4488689 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n621` (
  `n615` varchar(64) NOT NULL,
  `n041` varchar(128) NOT NULL,
  `n470` varchar(128) DEFAULT NULL,
  `n047` datetime DEFAULT NULL,
  `n629` datetime DEFAULT NULL,
  `n171` tinyint(1) DEFAULT 1,
  `n179` datetime DEFAULT current_timestamp(),
  `n084` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n615`),
  KEY `n049` (`n041`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n737` (
  `n703` varchar(15) NOT NULL,
  `n738` varchar(24) DEFAULT NULL,
  `n739` varchar(24) DEFAULT NULL,
  `n740` varchar(128) DEFAULT NULL,
  `n741` varchar(128) DEFAULT NULL,
  `n742` varchar(255) DEFAULT NULL,
  `n743` varchar(128) DEFAULT NULL,
  `n744` varchar(64) DEFAULT NULL,
  `n745` varchar(64) DEFAULT NULL,
  `n746` date DEFAULT NULL,
  `n747` varchar(32) DEFAULT NULL,
  `n748` date DEFAULT NULL,
  `n749` int(11) DEFAULT NULL,
  `n750` tinyint(4) NOT NULL DEFAULT 0,
  `n067` enum('v364','v365','v366') NOT NULL DEFAULT 'v367',
  `n751` datetime NOT NULL,
  `n046` datetime NOT NULL DEFAULT current_timestamp(),
  `n082` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n703`),
  KEY `n077` (`n067`),
  KEY `n752` (`n750`),
  KEY `n049` (`n742`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n753` (
  `n754` int(11) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n755` (
  `n114` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n756` int(10) unsigned NOT NULL,
  `n466` varchar(255) NOT NULL,
  `n757` varchar(32) DEFAULT NULL,
  `n116` datetime DEFAULT NULL,
  `n117` varchar(255) DEFAULT NULL,
  `n119` datetime DEFAULT NULL,
  `n120` varchar(255) DEFAULT NULL,
  `n121` varchar(32) DEFAULT NULL,
  `n053` varchar(32) NOT NULL DEFAULT 'v368',
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n114`),
  KEY `n758` (`n756`,`n119`),
  KEY `n466` (`n466`),
  KEY `n757` (`n757`),
  KEY `n116` (`n116`),
  CONSTRAINT `n759` FOREIGN KEY (`n756`) REFERENCES `n760` (`n756`) ON DELETE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=3488 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n760` (
  `n756` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n053` enum('v369','v370','v371') NOT NULL,
  `n761` varchar(64) DEFAULT NULL,
  `n762` enum('v372','v373','v374','v375','v376','v377','v378','v379','v380','v381') NOT NULL DEFAULT 'v382',
  `n369` varchar(128) DEFAULT NULL,
  `n097` varchar(128) DEFAULT NULL,
  `n763` varchar(128) DEFAULT NULL,
  `n764` varchar(128) DEFAULT NULL,
  `n024` varchar(128) DEFAULT NULL,
  `n765` int(10) unsigned DEFAULT NULL,
  `n167` varchar(16) DEFAULT NULL,
  `n128` varchar(64) DEFAULT NULL,
  `n766` varchar(255) DEFAULT NULL,
  `n767` varchar(32) DEFAULT NULL,
  `n768` varchar(128) DEFAULT NULL,
  `n067` enum('v383','v384','v385','v386','v387','v388') NOT NULL DEFAULT 'v389',
  `n389` datetime DEFAULT NULL,
  `n390` varchar(128) DEFAULT NULL,
  `n047` datetime DEFAULT NULL,
  `n769` varchar(128) DEFAULT NULL,
  `n770` varchar(64) DEFAULT NULL,
  `n771` date DEFAULT NULL,
  `n772` decimal(10,2) DEFAULT NULL,
  `n773` date DEFAULT NULL,
  `n774` varchar(255) DEFAULT NULL,
  `n082` text DEFAULT NULL,
  `n751` datetime DEFAULT NULL,
  `n179` datetime DEFAULT current_timestamp(),
  `n084` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n756`),
  UNIQUE KEY `n775` (`n053`,`n761`),
  KEY `n010` (`n097`),
  KEY `n776` (`n763`),
  KEY `n111` (`n167`),
  KEY `n777` (`n762`),
  KEY `n077` (`n067`),
  KEY `n778` (`n766`),
  KEY `n779` (`n765`),
  KEY `n133` (`n128`),
  KEY `n780` (`n768`),
  KEY `n781` (`n767`)
) ENGINE=InnoDB AUTO_INCREMENT=255249332 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n782` (
  `n669` int(10) unsigned NOT NULL,
  `n338` varchar(50) DEFAULT NULL,
  `n783` varchar(24) DEFAULT NULL,
  `n784` varchar(255) DEFAULT NULL,
  `n785` tinyint(1) NOT NULL DEFAULT 0,
  `n786` tinyint(1) NOT NULL DEFAULT 0,
  `n787` varchar(128) DEFAULT NULL,
  `n788` datetime DEFAULT NULL,
  `n084` datetime NOT NULL,
  PRIMARY KEY (`n669`),
  KEY `n789` (`n783`),
  KEY `n010` (`n338`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n790` (
  `n041` varchar(128) NOT NULL,
  `n258` char(64) NOT NULL,
  `n073` datetime NOT NULL,
  `n360` int(11) NOT NULL DEFAULT 0,
  `n606` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n041`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n791` (
  `n051` char(64) NOT NULL,
  `n792` int(11) NOT NULL,
  `n041` varchar(128) NOT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n107` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  KEY `n041` (`n041`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n793` (
  `n794` varchar(24) NOT NULL,
  `n192` varchar(64) NOT NULL,
  `n795` smallint(6) NOT NULL DEFAULT 100,
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  `n179` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`n794`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n796` (
  `n792` int(11) NOT NULL AUTO_INCREMENT,
  `n041` varchar(128) NOT NULL,
  `n470` varchar(128) NOT NULL,
  `n797` varchar(128) DEFAULT NULL,
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  `n179` timestamp NOT NULL DEFAULT current_timestamp(),
  `n798` varchar(64) DEFAULT NULL,
  `n799` datetime DEFAULT NULL,
  `n800` datetime DEFAULT NULL,
  PRIMARY KEY (`n792`),
  UNIQUE KEY `n041` (`n041`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n801` (
  `n802` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n803` int(10) unsigned NOT NULL,
  `n804` int(10) unsigned DEFAULT NULL,
  `n805` varchar(4000) NOT NULL,
  `n806` varchar(128) NOT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n802`),
  KEY `n807` (`n803`,`n802`),
  KEY `n077` (`n804`)
) ENGINE=InnoDB AUTO_INCREMENT=254 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n808` (
  `n809` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n803` int(10) unsigned NOT NULL,
  `n810` smallint(5) unsigned DEFAULT NULL,
  `n613` varchar(128) NOT NULL,
  `n811` smallint(5) unsigned NOT NULL DEFAULT 1,
  `n812` decimal(8,2) NOT NULL,
  `n813` decimal(10,2) NOT NULL,
  PRIMARY KEY (`n809`),
  KEY `n803` (`n803`)
) ENGINE=InnoDB AUTO_INCREMENT=5699 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n814` (
  `n815` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n803` int(10) unsigned NOT NULL,
  `n816` varchar(24) NOT NULL,
  `n817` decimal(10,2) NOT NULL,
  `n818` enum('v390','v391','v392','v393') NOT NULL DEFAULT 'v394',
  `n819` varchar(64) DEFAULT NULL,
  `n165` varchar(255) DEFAULT NULL,
  `n820` varchar(128) NOT NULL,
  `n821` datetime NOT NULL DEFAULT current_timestamp(),
  `n822` datetime DEFAULT NULL,
  `n823` varchar(128) DEFAULT NULL,
  `n824` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n815`),
  UNIQUE KEY `n825` (`n816`),
  KEY `n807` (`n803`),
  KEY `n826` (`n821`)
) ENGINE=InnoDB AUTO_INCREMENT=101 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n827` (
  `n828` int(10) unsigned NOT NULL,
  `n803` int(10) unsigned NOT NULL,
  `n829` varchar(20) NOT NULL,
  `n167` varchar(16) NOT NULL,
  `n830` varchar(16) NOT NULL,
  `n817` decimal(10,2) NOT NULL,
  PRIMARY KEY (`n828`,`n803`),
  UNIQUE KEY `n831` (`n803`),
  KEY `n111` (`n167`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n832` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n833` date NOT NULL,
  `n834` date NOT NULL,
  `n835` int(10) unsigned NOT NULL DEFAULT 0,
  `n836` decimal(12,2) NOT NULL DEFAULT 0.00,
  `n837` varchar(128) NOT NULL,
  `n838` datetime NOT NULL DEFAULT current_timestamp(),
  `n082` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  KEY `n839` (`n833`,`n834`),
  KEY `n840` (`n838`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n841` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n803` int(10) unsigned NOT NULL,
  `n842` varchar(20) DEFAULT NULL,
  `n843` varchar(20) DEFAULT NULL,
  `n844` varchar(128) DEFAULT NULL,
  `n845` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  KEY `n803` (`n803`)
) ENGINE=InnoDB AUTO_INCREMENT=6515 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n846` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n847` int(10) unsigned NOT NULL,
  `n848` int(10) unsigned NOT NULL,
  `n849` enum('v395','v396') NOT NULL,
  `n065` varchar(255) DEFAULT NULL,
  `n208` varchar(128) NOT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n850` (`n847`),
  UNIQUE KEY `n851` (`n848`),
  KEY `n181` (`n179`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n852` (
  `n803` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n829` varchar(20) NOT NULL,
  `n669` int(10) unsigned DEFAULT NULL,
  `n167` varchar(16) NOT NULL,
  `n853` enum('v397','v398') NOT NULL DEFAULT 'v399',
  `n830` enum('v400','v401','v402','v403','v404','v405') DEFAULT NULL,
  `n854` varchar(128) DEFAULT NULL,
  `n855` varchar(32) DEFAULT NULL,
  `n856` varchar(8) DEFAULT NULL,
  `n338` varchar(50) DEFAULT NULL,
  `n756` varchar(100) DEFAULT NULL,
  `n740` varchar(50) DEFAULT NULL,
  `n836` decimal(10,2) NOT NULL DEFAULT 0.00,
  `n067` enum('v406','v407','v408','v409','v410','v411','v412') NOT NULL DEFAULT 'v413',
  `n261` varchar(128) DEFAULT NULL,
  `n262` datetime DEFAULT NULL,
  `n518` datetime DEFAULT NULL,
  `n857` varchar(128) DEFAULT NULL,
  `n082` text DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`n803`),
  UNIQUE KEY `n829` (`n829`),
  KEY `n167` (`n167`),
  KEY `n067` (`n067`),
  KEY `n669` (`n669`),
  KEY `n830` (`n830`),
  KEY `n262` (`n262`)
) ENGINE=InnoDB AUTO_INCREMENT=5726 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n858` (
  `n859` varchar(96) NOT NULL,
  `n860` int(10) unsigned NOT NULL,
  PRIMARY KEY (`n859`),
  KEY `n861` (`n860`),
  CONSTRAINT `n861` FOREIGN KEY (`n860`) REFERENCES `n862` (`n860`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n862` (
  `n860` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n369` varchar(96) NOT NULL,
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  `n179` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`n860`),
  UNIQUE KEY `n539` (`n369`)
) ENGINE=InnoDB AUTO_INCREMENT=94 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n863` (
  `n864` varchar(220) NOT NULL,
  `n765` int(10) unsigned NOT NULL,
  PRIMARY KEY (`n864`),
  KEY `n865` (`n765`),
  CONSTRAINT `n865` FOREIGN KEY (`n765`) REFERENCES `n866` (`n765`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n866` (
  `n765` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n860` int(10) unsigned NOT NULL,
  `n794` varchar(24) NOT NULL,
  `n369` varchar(128) NOT NULL,
  `n867` tinyint(1) NOT NULL DEFAULT 0,
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  `n179` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`n765`),
  UNIQUE KEY `n868` (`n860`,`n369`),
  KEY `n777` (`n794`),
  KEY `n869` (`n867`),
  CONSTRAINT `n870` FOREIGN KEY (`n860`) REFERENCES `n862` (`n860`),
  CONSTRAINT `n871` FOREIGN KEY (`n794`) REFERENCES `n793` (`n794`)
) ENGINE=InnoDB AUTO_INCREMENT=340 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n872` (
  `n051` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `n873` char(40) NOT NULL COMMENT 'v414',
  `n171` tinyint(3) unsigned DEFAULT NULL COMMENT 'v415',
  `n669` int(10) unsigned NOT NULL,
  `n874` varchar(64) NOT NULL DEFAULT 'v416',
  `n875` varchar(190) NOT NULL DEFAULT 'v417' COMMENT 'v418',
  `n876` varchar(200) NOT NULL DEFAULT 'v419',
  `n877` varchar(16) NOT NULL DEFAULT 'v420',
  `n878` datetime DEFAULT NULL COMMENT 'v421',
  `n393` datetime NOT NULL COMMENT 'v422',
  `n107` datetime NOT NULL COMMENT 'v423',
  `n430` datetime DEFAULT NULL,
  `n879` int(10) unsigned NOT NULL DEFAULT 1 COMMENT 'v424',
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n880` (`n873`,`n171`),
  KEY `n881` (`n669`),
  KEY `n212` (`n171`,`n393`)
) ENGINE=InnoDB AUTO_INCREMENT=1310 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='v425';
CREATE TABLE `n882` (
  `n051` tinyint(3) unsigned NOT NULL COMMENT 'v426',
  `n226` tinyint(3) unsigned NOT NULL DEFAULT 0 COMMENT 'v427',
  `n311` char(64) DEFAULT NULL COMMENT 'v428',
  `n883` char(4) DEFAULT NULL COMMENT 'v429',
  `n799` datetime DEFAULT NULL,
  `n884` varchar(128) DEFAULT NULL,
  `n800` datetime DEFAULT NULL,
  `n885` char(64) DEFAULT NULL COMMENT 'v430',
  `n886` datetime DEFAULT NULL,
  `n887` int(10) unsigned DEFAULT NULL COMMENT 'v431',
  `n888` varchar(16) NOT NULL DEFAULT 'v432' COMMENT 'v433',
  `n889` int(10) unsigned NOT NULL DEFAULT 20 COMMENT 'v434',
  `n890` varchar(128) NOT NULL DEFAULT 'v435',
  `n891` datetime DEFAULT NULL COMMENT 'v436',
  `n892` datetime DEFAULT NULL COMMENT 'v437',
  `n405` text DEFAULT NULL,
  `n084` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='v438';
CREATE TABLE `n893` (
  `n810` smallint(5) unsigned NOT NULL AUTO_INCREMENT,
  `n894` varchar(32) DEFAULT NULL,
  `n613` varchar(128) NOT NULL,
  `n812` decimal(8,2) NOT NULL,
  `n191` enum('v439','v440','v441') NOT NULL DEFAULT 'v442',
  `n024` varchar(64) DEFAULT NULL,
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (`n810`),
  KEY `n895` (`n024`)
) ENGINE=InnoDB AUTO_INCREMENT=31 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n896` (
  `n669` int(10) unsigned NOT NULL,
  `n067` enum('v443','v444','v445','v446','v447') NOT NULL DEFAULT 'v448',
  `n167` varchar(16) NOT NULL,
  `n168` varchar(45) DEFAULT NULL,
  `n169` varchar(128) DEFAULT NULL,
  `n897` bigint(20) DEFAULT NULL,
  `n898` varchar(128) DEFAULT NULL,
  `n899` varchar(255) DEFAULT NULL,
  `n403` varchar(128) NOT NULL,
  `n068` datetime NOT NULL DEFAULT current_timestamp(),
  `n787` varchar(128) DEFAULT NULL,
  `n788` datetime DEFAULT NULL,
  `n900` varchar(500) DEFAULT NULL,
  `n901` int(10) unsigned DEFAULT NULL,
  `n902` char(17) DEFAULT NULL,
  `n105` varchar(128) DEFAULT NULL,
  `n024` varchar(128) DEFAULT NULL,
  `n764` varchar(128) DEFAULT NULL,
  `n765` int(10) unsigned DEFAULT NULL,
  `n903` varchar(128) DEFAULT NULL,
  `n904` datetime DEFAULT NULL,
  `n905` varchar(45) DEFAULT NULL,
  `n906` datetime DEFAULT NULL,
  `n907` int(10) unsigned DEFAULT NULL,
  `n908` int(10) unsigned DEFAULT NULL,
  `n909` datetime DEFAULT NULL,
  `n910` datetime DEFAULT NULL,
  `n405` varchar(500) DEFAULT NULL,
  PRIMARY KEY (`n669`),
  KEY `n077` (`n067`),
  KEY `n111` (`n167`),
  KEY `n078` (`n068`),
  KEY `n911` (`n067`,`n910`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n912` (
  `n887` smallint(5) unsigned NOT NULL,
  `n913` varchar(128) NOT NULL,
  PRIMARY KEY (`n887`,`n913`),
  KEY `n913` (`n913`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n914` (
  `n167` varchar(16) NOT NULL,
  `n887` smallint(5) unsigned NOT NULL,
  PRIMARY KEY (`n167`),
  KEY `n887` (`n887`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n915` (
  `n887` smallint(5) unsigned NOT NULL AUTO_INCREMENT,
  `n369` varchar(64) NOT NULL,
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n887`),
  UNIQUE KEY `n369` (`n369`)
) ENGINE=InnoDB AUTO_INCREMENT=27 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n916` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n167` varchar(16) NOT NULL,
  `n917` varchar(128) NOT NULL,
  `n918` varchar(128) DEFAULT NULL,
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (`n051`),
  KEY `n167` (`n167`)
) ENGINE=InnoDB AUTO_INCREMENT=41 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n919` (
  `n920` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n669` int(10) unsigned NOT NULL,
  `n921` varchar(128) NOT NULL COMMENT 'v449',
  `n922` datetime NOT NULL DEFAULT current_timestamp(),
  `n923` varchar(512) DEFAULT NULL COMMENT 'v450',
  `n689` int(10) unsigned DEFAULT NULL COMMENT 'v451',
  `n165` varchar(1000) DEFAULT NULL,
  `n802` int(10) unsigned DEFAULT NULL COMMENT 'v452',
  PRIMARY KEY (`n920`),
  KEY `n924` (`n669`),
  KEY `n925` (`n922`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n926` (
  `n927` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n041` varchar(128) NOT NULL COMMENT 'v453',
  `n311` char(64) NOT NULL COMMENT 'v454',
  `n066` varchar(32) NOT NULL COMMENT 'v455',
  `n192` varchar(128) DEFAULT NULL COMMENT 'v456',
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n208` varchar(128) DEFAULT NULL,
  `n073` datetime DEFAULT NULL COMMENT 'v457',
  `n421` datetime DEFAULT NULL,
  `n928` varchar(45) DEFAULT NULL,
  `n263` datetime DEFAULT NULL,
  `n264` varchar(128) DEFAULT NULL,
  `n929` text DEFAULT NULL COMMENT 'v458',
  `n930` varchar(16) DEFAULT NULL COMMENT 'v459',
  PRIMARY KEY (`n927`),
  UNIQUE KEY `n931` (`n311`),
  KEY `n932` (`n041`)
) ENGINE=InnoDB AUTO_INCREMENT=28 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n933` (
  `n934` date NOT NULL,
  `n935` bigint(20) unsigned NOT NULL DEFAULT 0,
  `n936` int(10) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`n934`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n937` (
  `n938` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n669` int(10) unsigned DEFAULT NULL,
  `n803` int(10) unsigned DEFAULT NULL,
  `n939` varchar(255) NOT NULL,
  `n940` varchar(255) DEFAULT NULL,
  `n941` varchar(100) DEFAULT NULL,
  `n942` int(10) unsigned DEFAULT NULL,
  `n943` varchar(128) DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n938`),
  KEY `n669` (`n669`),
  KEY `n803` (`n803`)
) ENGINE=InnoDB AUTO_INCREMENT=389 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n944` (
  `n945` tinyint(3) unsigned NOT NULL AUTO_INCREMENT,
  `n369` varchar(64) NOT NULL,
  `n795` tinyint(3) unsigned DEFAULT 0,
  `n171` tinyint(1) DEFAULT 1,
  `n088` enum('v460','v461','v462') NOT NULL DEFAULT 'v463' COMMENT 'v464',
  PRIMARY KEY (`n945`)
) ENGINE=InnoDB AUTO_INCREMENT=43 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n946` (
  `n802` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n669` int(10) unsigned NOT NULL,
  `n947` varchar(128) NOT NULL,
  `n948` varchar(128) DEFAULT NULL,
  `n805` text NOT NULL,
  `n949` tinyint(1) NOT NULL DEFAULT 0,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n950` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n802`),
  KEY `n669` (`n669`),
  KEY `n950` (`n950`)
) ENGINE=InnoDB AUTO_INCREMENT=18854 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n951` (
  `n952` varchar(255) NOT NULL,
  `n669` int(11) NOT NULL,
  `n490` enum('v465','v466') NOT NULL,
  `n179` timestamp NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n952`),
  KEY `n669` (`n669`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n953` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n669` int(10) unsigned NOT NULL,
  `n842` varchar(20) DEFAULT NULL,
  `n843` varchar(20) DEFAULT NULL,
  `n844` varchar(128) DEFAULT NULL,
  `n845` datetime NOT NULL DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  KEY `n669` (`n669`)
) ENGINE=InnoDB AUTO_INCREMENT=36497 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n954` (
  `n669` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n890` varchar(128) NOT NULL,
  `n955` varchar(128) DEFAULT NULL,
  `n956` varchar(128) DEFAULT NULL,
  `n167` varchar(16) DEFAULT NULL,
  `n957` varchar(128) DEFAULT NULL,
  `n128` varchar(64) DEFAULT NULL,
  `n958` varchar(128) DEFAULT NULL,
  `n338` varchar(50) DEFAULT NULL,
  `n756` varchar(100) DEFAULT NULL,
  `n907` int(10) unsigned DEFAULT NULL,
  `n762` varchar(20) DEFAULT NULL,
  `n945` tinyint(3) unsigned DEFAULT NULL,
  `n959` enum('v467','v468','v469','v470') NOT NULL DEFAULT 'v471',
  `n067` enum('v472','v473','v474','v475','v476','v477','v478') NOT NULL DEFAULT 'v479',
  `n960` varchar(200) NOT NULL,
  `n805` text DEFAULT NULL,
  `n961` varchar(128) DEFAULT NULL,
  `n962` smallint(5) unsigned DEFAULT NULL,
  `n963` datetime DEFAULT NULL,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n084` datetime NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `n430` datetime DEFAULT NULL,
  `n964` enum('v480','v481','v482','v483') NOT NULL DEFAULT 'v484',
  `n965` tinyint(1) NOT NULL DEFAULT 1,
  `n966` datetime DEFAULT NULL,
  `n967` varchar(128) DEFAULT NULL,
  `n968` varchar(500) DEFAULT NULL,
  `n969` enum('v485','v486','v487','v488','v489') DEFAULT NULL COMMENT 'v490',
  `n970` varchar(2000) DEFAULT NULL,
  `n971` varchar(128) DEFAULT NULL,
  PRIMARY KEY (`n669`),
  KEY `n067` (`n067`),
  KEY `n167` (`n167`),
  KEY `n961` (`n961`),
  KEY `n890` (`n890`),
  KEY `n338` (`n338`),
  KEY `n179` (`n179`),
  KEY `n962` (`n962`),
  KEY `n972` (`n907`),
  KEY `n973` (`n762`),
  KEY `n966` (`n966`),
  KEY `n974` (`n956`)
) ENGINE=InnoDB AUTO_INCREMENT=17582 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n975` (
  `n976` bigint(20) NOT NULL COMMENT 'v491',
  `n977` varchar(24) NOT NULL COMMENT 'v492',
  `n978` varchar(128) DEFAULT NULL COMMENT 'v493',
  `n979` varchar(64) DEFAULT NULL,
  `n980` varchar(64) DEFAULT NULL COMMENT 'v494',
  `n981` tinyint(1) NOT NULL DEFAULT 0 COMMENT 'v495',
  `n046` datetime NOT NULL COMMENT 'v496',
  `n047` datetime NOT NULL,
  PRIMARY KEY (`n976`,`n977`),
  KEY `n048` (`n047`),
  KEY `n982` (`n981`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='v497';
CREATE TABLE `n983` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n728` varchar(128) NOT NULL,
  `n984` bigint(20) NOT NULL,
  `n616` varchar(128) NOT NULL,
  `n617` datetime DEFAULT current_timestamp(),
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n985` (`n728`,`n984`),
  KEY `n986` (`n728`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE `n987` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n988` int(10) unsigned NOT NULL,
  `n067` enum('v498','v499','v500','v501') NOT NULL DEFAULT 'v502',
  `n360` int(11) NOT NULL DEFAULT 0,
  `n179` datetime NOT NULL DEFAULT current_timestamp(),
  `n989` datetime DEFAULT NULL,
  `n990` datetime DEFAULT NULL,
  `n405` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n991` (`n988`),
  KEY `n077` (`n067`)
) ENGINE=InnoDB AUTO_INCREMENT=638 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
CREATE TABLE `n992` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n976` bigint(20) DEFAULT NULL,
  `n993` varchar(128) DEFAULT NULL COMMENT 'v503',
  `n984` bigint(20) NOT NULL,
  `n994` varchar(128) NOT NULL,
  `n995` varchar(32) NOT NULL,
  `n996` varchar(64) NOT NULL,
  `n997` varchar(64) NOT NULL,
  `n041` varchar(128) NOT NULL,
  `n998` enum('v504','v505') NOT NULL DEFAULT 'v506',
  `n312` varchar(128) DEFAULT NULL,
  `n489` varchar(64) NOT NULL DEFAULT 'v507',
  `n999` varchar(255) DEFAULT NULL,
  `n261` varchar(128) NOT NULL,
  `n262` datetime NOT NULL DEFAULT current_timestamp(),
  `n073` datetime DEFAULT NULL,
  `n966` datetime DEFAULT NULL,
  `n1000` datetime DEFAULT NULL,
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n1001` (`n976`),
  KEY `n049` (`n041`),
  KEY `n1002` (`n261`),
  KEY `n1003` (`n984`),
  KEY `n1004` (`n966`),
  KEY `n1005` (`n073`),
  KEY `n1006` (`n312`),
  KEY `n1007` (`n993`)
) ENGINE=InnoDB AUTO_INCREMENT=2507459 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE `n1008` (
  `n051` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `n984` bigint(20) NOT NULL,
  `n994` varchar(128) NOT NULL,
  `n1009` enum('v508','v509','v510') NOT NULL DEFAULT 'v511',
  `n171` tinyint(1) NOT NULL DEFAULT 1,
  `n1000` datetime DEFAULT NULL,
  `n179` datetime DEFAULT current_timestamp(),
  PRIMARY KEY (`n051`),
  UNIQUE KEY `n984` (`n984`)
) ENGINE=InnoDB AUTO_INCREMENT=5407 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
CREATE TABLE `n012` (
  `n1010` int(10) unsigned NOT NULL,
  `n369` varchar(64) DEFAULT NULL,
  `n1011` varchar(64) DEFAULT NULL,
  `n1012` varchar(128) DEFAULT NULL,
  `n1013` varchar(512) DEFAULT NULL,
  `n167` varchar(16) DEFAULT NULL,
  `n1014` enum('v512','v513','v514','v515') DEFAULT NULL,
  `n1015` varchar(16) DEFAULT NULL,
  `n764` varchar(128) DEFAULT NULL,
  `n024` varchar(128) DEFAULT NULL,
  `n097` varchar(128) DEFAULT NULL,
  `n763` varchar(128) DEFAULT NULL,
  `n1016` varchar(128) DEFAULT NULL,
  `n1017` varchar(64) DEFAULT NULL,
  `n1018` varchar(128) DEFAULT NULL,
  `n768` varchar(128) DEFAULT NULL,
  `n1019` datetime DEFAULT NULL,
  `n1020` datetime DEFAULT NULL,
  `n1021` datetime DEFAULT NULL,
  `n1022` varchar(128) DEFAULT NULL,
  `n1023` tinyint(1) DEFAULT NULL,
  `n1024` tinyint(1) DEFAULT NULL,
  `n1000` datetime DEFAULT NULL,
  PRIMARY KEY (`n1010`),
  KEY `n010` (`n097`),
  KEY `n111` (`n167`),
  KEY `n1025` (`n768`),
  KEY `n777` (`n1014`),
  KEY `n1026` (`n1021`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
