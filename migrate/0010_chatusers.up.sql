CREATE TABLE `chatusers` (
`chatuserid` VARCHAR(36) NOT NULL,
`username` VARCHAR(128) NOT NULL,
`type` VARCHAR(12) NOT NULL,
`alias` VARCHAR(128),
`nickname` VARCHAR(128) NOT NULL,
`avatar` text,
`ext` text,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`chatuserid`),
UNIQUE KEY (`type`, `username`),
INDEX `nickname_index` (`nickname`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
