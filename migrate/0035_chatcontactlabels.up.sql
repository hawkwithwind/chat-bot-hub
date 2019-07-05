CREATE TABLE `chatcontactlabels` (
`chatcontactlabelid` VARCHAR(36) NOT NULL,
`botid` VARCHAR(36) NOT NULL,
`labelid` INTEGER NOT NULL,
`label` VARCHAR(128) NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`chatcontactlabelid`),
UNIQUE KEY (`botid`, `labelid`, `label`),
INDEX `botid_index` (`botid`),
INDEX `botid_labelid` (`botid`, `labelid`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
