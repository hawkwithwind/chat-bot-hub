CREATE TABLE `chatcontacts` (
`chatcontactid` VARCHAR(36) NOT NULL,
`botid` VARCHAR(36) NOT NULL,
`chatuserid` VARCHAR(36) NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`chatcontactid`),
UNIQUE KEY (`botid`, `chatuserid`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
