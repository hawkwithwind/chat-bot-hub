CREATE TABLE `devices` (
`deviceid` VARCHAR(36) NOT NULL,
`devicename` VARCHAR(36) NOT NULL,
`accountid` VARCHAR(36) NOT NULL,
`chatbottype` VARCHAR(36) NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`deviceid`),
UNIQUE KEY (`devicename`),
INDEX `devicename_index` (`devicename`),
INDEX `accountid_index` (`accountid`),
INDEX `chatbottype_index` (`chatbottype`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
