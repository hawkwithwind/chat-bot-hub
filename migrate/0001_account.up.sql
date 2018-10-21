CREATE TABLE `accounts` (
`accountid` VARCHAR(36) NOT NULL,
`accountname` VARCHAR(36) NOT NULL,
`email` VARCHAR(128),
`secret` VARCHAR(128) NOT NULL,
`createat` DATETIME DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`accountid`),
UNIQUE KEY (`accountname`),
UNIQUE KEY (`email`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

