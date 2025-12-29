CREATE TABLE `admin` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT COMMENT '自增ID',
    `username` VARCHAR(16) NOT NULL COMMENT '用户名',
    `password` VARCHAR(255) NOT NULL COMMENT '密码',
    `type` TINYINT NOT NULL DEFAULT 1 COMMENT '类型 1-普通管理员 2-超级管理员',
    `created_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='管理员表';

CREATE TABLE `survey` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT COMMENT '自增ID',
    `admin_id` BIGINT UNSIGNED NOT NULL COMMENT '所属管理员ID',
    `title` VARCHAR(64) NOT NULL COMMENT '标题',
    `type` TINYINT NOT NULL COMMENT '类型 1-问卷 2-投票',
    `path` VARCHAR(64) NOT NULL COMMENT '访问路径',
    `schema` JSON NOT NULL COMMENT '结构',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态 1-未发布 2-已发布',
    `created_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    `deleted_at` BIGINT NOT NULL DEFAULT 0 COMMENT '删除时间 (软删除)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_path_deleted_at` (`path`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='问卷表';

CREATE TABLE `result` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT COMMENT '自增ID',
    `survey_id` BIGINT UNSIGNED NOT NULL COMMENT '问卷ID',
    `username` VARCHAR(16) NOT NULL DEFAULT '' COMMENT '用户名',
    `data` JSON NOT NULL COMMENT '答卷内容',
    `created_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    PRIMARY KEY (`id`),
    INDEX `idx_survey_id_username_created_at` (`survey_id`, `username`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='答卷表';

CREATE TABLE `stats` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT COMMENT '自增ID',
    `survey_id` BIGINT UNSIGNED NOT NULL COMMENT '问卷ID',
    `question_id` VARCHAR(16) NOT NULL COMMENT '题目ID',
    `option_id` VARCHAR(16) NOT NULL COMMENT '选项ID',
    `count` INT NOT NULL DEFAULT 0 COMMENT '数量',
    `created_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `updated_at` TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_survey_id_question_id_option_id` (`survey_id`, `question_id`, `option_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='统计表';