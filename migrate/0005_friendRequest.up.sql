CREATE TABLE `friendrequests` (
`friendrequestid` VARCHAR(36) NOT NULL,
`botid` VARCHAR(36) NOT NULL,
`login` VARCHAR(128) NOT NULL,
`requestlogin` VARCHAR(128) NOT NULL,
`requestbody` TEXT,
`status` VARCHAR(12) NOT NULL DEFAULT 'NEW',
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
PRIMARY KEY (`friendrequestid`),
INDEX `login_index` (`login`),
INDEX `requestlogin_index` (`requestlogin`),
INDEX `status_index` (`status`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
