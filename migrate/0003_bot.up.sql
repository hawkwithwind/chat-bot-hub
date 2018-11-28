CREATE TABLE `bots` (
`botid` VARCHAR(36) NOT NULL,
`accountid` VARCHAR(36) NOT NULL,
`botname` VARCHAR(128) NOT NULL,
`login` VARCHAR(128) NOT NULL,
`chatbottype` VARCHAR(36) NOT NULL,
`logininfo` TEXT,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`botid`),
UNIQUE KEY (`accountid`, `botname`),
UNIQUE KEY (`accountid`, `login`),
INDEX `login_index` (`login`),
INDEX `chatbottype_index` (`chatbottype`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
