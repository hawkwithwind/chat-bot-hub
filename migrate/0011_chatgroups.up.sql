CREATE TABLE `chatgroups` (
`chatgroupid` VARCHAR(36) NOT NULL,
`groupname` VARCHAR(128) NOT NULL,
`type` VARCHAR(12) NOT NULL,
`alias` VARCHAR(128),
`nickname` VARCHAR(128) NOT NULL,
`owner` VARCHAR(36) NOT NULL,
`membercount` INT NOT NULL,
`maxmembercount` INT NOT NULL,
`avatar` text,
`ext` text,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`chatgroupid`),
UNIQUE KEY (`type`, `groupname`),
INDEX `nickname_index` (`nickname`),
INDEX `owner_index` (`owner`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
