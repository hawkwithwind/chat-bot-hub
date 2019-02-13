CREATE TABLE `chatgroupmembers` (
`chatgroupmemberid` VARCHAR(36) NOT NULL,
`chatgroupid` VARCHAR(36) NOT NULL,
`chatmemberid` VARCHAR(36) NOT NULL,
`invitedby` VARCHAR(36),
`attendance` INT,
`groupnickname` VARCHAR(128),
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`chatgroupmemberid`),
UNIQUE (`chatgroupid`, `chatmemberid`),
INDEX `chatgroupid_index` (`chatgroupid`),
INDEX `chatmemberid_index` (`chatmemberid`),
INDEX `chatmemberid_attendance_index` (`chatmemberid`, `attendance`),
INDEX `chatmemberid_invitedby_index` (`chatmemberid`, `invitedby`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
