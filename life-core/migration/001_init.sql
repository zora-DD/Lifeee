CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '用户ID',
    wx_open_id VARCHAR(64) NOT NULL COMMENT '小程序OpenID',
    wx_union_id VARCHAR(64) NULL COMMENT '微信UnionID',
    nickname VARCHAR(64) NOT NULL COMMENT '昵称',
    avatar_url VARCHAR(512) NULL COMMENT '头像URL',
    avatar_emoji VARCHAR(16) NULL COMMENT '随机头像Emoji',
    avatar_color VARCHAR(16) NULL COMMENT '头像背景色',
    signature VARCHAR(160) NULL COMMENT '签名',
    application_note VARCHAR(200) NULL COMMENT '访问申请身份备注',
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/approved/rejected/disabled',
    role VARCHAR(16) NOT NULL DEFAULT 'user' COMMENT 'user/admin',
    last_login_at DATETIME(3) NULL COMMENT '最近登录时间',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_users_wx_open_id (wx_open_id),
    UNIQUE KEY uk_users_wx_union_id (wx_union_id),
    KEY idx_users_status (status)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='用户';

CREATE TABLE IF NOT EXISTS tasks (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '任务ID',
    title VARCHAR(120) NOT NULL COMMENT '标题',
    description TEXT NULL COMMENT '任务说明',
    status VARCHAR(16) NOT NULL DEFAULT 'todo' COMMENT 'todo/done',
    priority VARCHAR(16) NOT NULL DEFAULT 'medium' COMMENT 'high/medium/low',
    category VARCHAR(32) NOT NULL DEFAULT '其他' COMMENT '任务分类',
    due_at DATETIME(3) NULL COMMENT '截止时间',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '创建人ID，逻辑关联users.id',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    KEY idx_tasks_list (deleted_at, status, priority, due_at),
    KEY idx_tasks_creator (created_by, created_at)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='任务';

CREATE TABLE IF NOT EXISTS activity_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '生活记录ID',
    content VARCHAR(2000) NOT NULL COMMENT '文字或语音转写内容',
    occurred_at DATETIME(3) NOT NULL COMMENT '实际发生时间',
    duration_minutes SMALLINT UNSIGNED NULL COMMENT '持续分钟数',
    input_type VARCHAR(16) NOT NULL DEFAULT 'text' COMMENT 'text/voice',
    audio_url VARCHAR(512) NULL COMMENT '原始语音URL，预留',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '记录人ID，逻辑关联users.id',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    KEY idx_activity_logs_timeline (created_by, deleted_at, occurred_at DESC, id DESC),
    KEY idx_activity_logs_stats (deleted_at, occurred_at, created_by)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='生活流水记录';

CREATE TABLE IF NOT EXISTS bills (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '账单ID',
    type VARCHAR(16) NOT NULL DEFAULT 'expense' COMMENT 'expense支出/income收入',
    title VARCHAR(120) NOT NULL COMMENT '账单标题',
    amount DECIMAL(12,2) NOT NULL COMMENT '金额，始终保存正数',
    category VARCHAR(32) NOT NULL COMMENT '分类',
    occurred_at DATETIME(3) NOT NULL COMMENT '消费或收入发生时间',
    note VARCHAR(1000) NULL COMMENT '备注',
    input_type VARCHAR(16) NOT NULL DEFAULT 'manual' COMMENT 'manual/text/voice',
    raw_text VARCHAR(2000) NULL COMMENT '用户原始输入文本，不保存音频Base64',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '记账人ID，逻辑关联users.id',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    KEY idx_bills_list (deleted_at, occurred_at DESC, id DESC),
    KEY idx_bills_creator (created_by, occurred_at DESC),
    KEY idx_bills_category (category, occurred_at DESC),
    KEY idx_bills_type (type, occurred_at DESC)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='公开账单记录';

CREATE TABLE IF NOT EXISTS restaurants (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '吃喝地点ID',
    name VARCHAR(120) NOT NULL COMMENT '名称',
    category VARCHAR(32) NOT NULL COMMENT '吃喝分类',
    cuisine VARCHAR(64) NULL COMMENT '细分类',
    preference VARCHAR(16) NOT NULL DEFAULT 'good' COMMENT 'okay/good/love/must',
    address VARCHAR(255) NULL COMMENT '地址',
    latitude DECIMAL(10,7) NULL COMMENT '纬度',
    longitude DECIMAL(10,7) NULL COMMENT '经度',
    business_hours VARCHAR(120) NULL COMMENT '营业时间文本',
    features VARCHAR(500) NULL COMMENT '店铺特点',
    cover_url VARCHAR(512) NULL COMMENT '封面URL',
    cover_object_key VARCHAR(255) NULL COMMENT '对象存储Key',
    visit_count INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '消费次数汇总',
    total_spent DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '累计消费',
    last_visited_at DATE NULL COMMENT '最近消费日期',
    last_amount DECIMAL(10,2) NULL COMMENT '最近一次金额',
    last_meal VARCHAR(500) NULL COMMENT '最近吃喝内容',
    last_note VARCHAR(1000) NULL COMMENT '最近一次备注',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '创建人ID，逻辑关联users.id',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    KEY idx_restaurants_list (deleted_at, category, last_visited_at),
    KEY idx_restaurants_creator (created_by, created_at),
    KEY idx_restaurants_preference (preference)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='吃喝地点';

CREATE TABLE IF NOT EXISTS restaurant_consumptions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '消费记录ID',
    restaurant_id BIGINT UNSIGNED NOT NULL COMMENT '吃喝地点ID，逻辑关联restaurants.id',
    consumed_by BIGINT UNSIGNED NOT NULL COMMENT '记录人ID，逻辑关联users.id',
    consumed_at DATE NOT NULL COMMENT '消费日期',
    amount DECIMAL(10,2) NULL COMMENT '本次金额，允许未知',
    meal VARCHAR(500) NULL COMMENT '吃喝内容',
    note VARCHAR(1000) NULL COMMENT '备注',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    KEY idx_consumptions_restaurant_date (restaurant_id, deleted_at, consumed_at DESC, id DESC),
    KEY idx_consumptions_user (consumed_by, created_at)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='吃喝消费记录';

CREATE TABLE IF NOT EXISTS home_dishes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '家庭菜品ID',
    name VARCHAR(120) NOT NULL COMMENT '菜名',
    category VARCHAR(32) NOT NULL COMMENT '菜品分类',
    ingredients VARCHAR(1000) NOT NULL DEFAULT '' COMMENT '主要配料',
    steps TEXT NOT NULL COMMENT '制作步骤',
    difficulty VARCHAR(16) NOT NULL DEFAULT 'normal' COMMENT 'easy/normal/hard',
    estimated_minutes INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '预计制作分钟数',
    cover_url VARCHAR(512) NULL COMMENT '封面URL',
    cover_object_key VARCHAR(255) NULL COMMENT '封面对象Key',
    experience VARCHAR(2000) NULL COMMENT '制作心得',
    rating TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '综合评价1-5',
    cook_count INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '制作次数',
    last_cooked_at DATE NULL COMMENT '最近制作日期',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '创建人ID',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_home_dishes_list (deleted_at, category, last_cooked_at),
    KEY idx_home_dishes_creator (created_by, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='家庭菜品';

CREATE TABLE IF NOT EXISTS home_cooking_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '制作记录ID',
    dish_id BIGINT UNSIGNED NOT NULL COMMENT '家庭菜品ID',
    cooked_by BIGINT UNSIGNED NOT NULL COMMENT '制作人ID',
    cooked_at DATE NOT NULL COMMENT '制作日期',
    duration_minutes INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '实际制作分钟数',
    rating TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '本次评价1-5',
    note VARCHAR(1000) NULL COMMENT '本次心得',
    photo_url VARCHAR(512) NULL COMMENT '本次照片URL',
    photo_object_key VARCHAR(255) NULL COMMENT '照片对象Key',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_home_cooking_dish_date (dish_id, deleted_at, cooked_at DESC),
    KEY idx_home_cooking_user (cooked_by, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='家庭制作记录';

CREATE TABLE IF NOT EXISTS snacks (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '零食ID',
    name VARCHAR(120) NOT NULL COMMENT '零食名称',
    brand VARCHAR(120) NOT NULL DEFAULT '' COMMENT '品牌',
    purchase_place VARCHAR(255) NOT NULL DEFAULT '' COMMENT '购买地点或渠道',
    category VARCHAR(32) NOT NULL COMMENT '零食分类',
    status VARCHAR(16) NOT NULL DEFAULT 'wishlist' COMMENT 'wishlist/eaten/frequent/avoid',
    cover_url VARCHAR(512) NULL COMMENT '封面URL',
    cover_object_key VARCHAR(255) NULL COMMENT '封面对象Key',
    note VARCHAR(2000) NULL COMMENT '口味感受',
    memory_tag VARCHAR(64) NULL COMMENT '回忆标签',
    tasting_count INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '品尝次数',
    last_tasted_at DATE NULL COMMENT '最近品尝日期',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '创建人ID',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_snacks_list (deleted_at, status, category, updated_at),
    KEY idx_snacks_creator (created_by, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='零食收藏';

CREATE TABLE IF NOT EXISTS snack_tasting_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '品尝记录ID',
    snack_id BIGINT UNSIGNED NOT NULL COMMENT '零食ID',
    tasted_by BIGINT UNSIGNED NOT NULL COMMENT '记录人ID',
    tasted_at DATE NOT NULL COMMENT '品尝日期',
    note VARCHAR(1000) NULL COMMENT '本次感受',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id),
    KEY idx_snack_tasting_date (snack_id, deleted_at, tasted_at DESC),
    KEY idx_snack_tasting_user (tasted_by, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='零食品尝记录';

CREATE TABLE IF NOT EXISTS places (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '地点ID',
    name VARCHAR(120) NOT NULL COMMENT '地点名称',
    category VARCHAR(32) NOT NULL COMMENT '地点分类',
    icon VARCHAR(16) NULL COMMENT '分类图标',
    visited TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否去过',
    visit_count INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '去过次数',
    last_visited_at DATE NULL COMMENT '最近到访日期',
    feature VARCHAR(1000) NULL COMMENT '地点特色',
    note VARCHAR(2000) NULL COMMENT '备注',
    address VARCHAR(255) NULL COMMENT '地址或集合点',
    best_time VARCHAR(120) NULL COMMENT '适合时间',
    distance_km INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '以上海为起点的参考距离（公里）',
    drive_duration_minutes INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '参考自驾时长（分钟）',
    guide_url VARCHAR(512) NULL COMMENT '微信公众号攻略文章地址',
    cover_url VARCHAR(512) NULL COMMENT '封面URL',
    cover_object_key VARCHAR(255) NULL COMMENT '对象存储Key',
    latitude DECIMAL(10,7) NULL COMMENT '纬度，预留',
    longitude DECIMAL(10,7) NULL COMMENT '经度，预留',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '创建人ID，逻辑关联users.id',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    KEY idx_places_list (deleted_at, category, visited, created_at),
    KEY idx_places_creator (created_by, created_at)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='地点';

CREATE TABLE IF NOT EXISTS favorites (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '收藏ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '收藏用户ID，逻辑关联users.id',
    target_type VARCHAR(16) NOT NULL COMMENT 'task/restaurant/place',
    target_id BIGINT UNSIGNED NOT NULL COMMENT '目标内容ID，逻辑多态关联',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_favorites_user_target (user_id, target_type, target_id),
    KEY idx_favorites_user_time (user_id, created_at),
    KEY idx_favorites_target (target_type, target_id)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='收藏';

CREATE TABLE IF NOT EXISTS media_assets (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '文件ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '上传人ID，逻辑关联users.id',
    biz_type VARCHAR(32) NOT NULL COMMENT 'avatar/restaurant/place/activity_log',
    object_key VARCHAR(255) NOT NULL COMMENT '对象存储Key',
    file_url VARCHAR(512) NOT NULL COMMENT '访问URL',
    mime_type VARCHAR(64) NOT NULL COMMENT '真实MIME类型',
    file_size BIGINT UNSIGNED NOT NULL COMMENT '文件字节数',
    width INT UNSIGNED NULL COMMENT '图片宽度',
    height INT UNSIGNED NULL COMMENT '图片高度',
    status VARCHAR(16) NOT NULL DEFAULT 'active' COMMENT 'active/deleted',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_media_assets_object_key (object_key),
    KEY idx_media_assets_user_time (user_id, created_at),
    KEY idx_media_assets_status (status, created_at)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='上传文件';

CREATE TABLE IF NOT EXISTS memories (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '时光记录ID',
    memory_date DATE NOT NULL COMMENT '记录日期，不完整日期使用当月或当年第一天',
    date_precision VARCHAR(8) NOT NULL DEFAULT 'day' COMMENT 'day/month/year',
    place VARCHAR(255) NULL COMMENT '地点',
    title VARCHAR(160) NOT NULL COMMENT '标题',
    description VARCHAR(1000) NULL COMMENT '描述',
    source_key VARCHAR(96) NULL COMMENT '迁移来源唯一标识',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '创建人ID，逻辑关联users.id',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_memories_source_key (source_key),
    KEY idx_memories_timeline (deleted_at, memory_date, id),
    KEY idx_memories_creator (created_by, created_at)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='时光相册记录';

CREATE TABLE IF NOT EXISTS memory_photos (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '时光图片ID',
    memory_id BIGINT UNSIGNED NOT NULL COMMENT '时光记录ID，逻辑关联memories.id',
    media_asset_id BIGINT UNSIGNED NOT NULL COMMENT '媒体ID，逻辑关联media_assets.id',
    object_key VARCHAR(255) NOT NULL COMMENT '媒体对象Key',
    source_name VARCHAR(255) NULL COMMENT '迁移前文件名',
    alt_text VARCHAR(500) NULL COMMENT '图片说明',
    width INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '宽度',
    height INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '高度',
    sort_order INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '同一记录内排序',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_memory_photos_memory_order (memory_id, sort_order),
    KEY idx_memory_photos_media (media_asset_id),
    KEY idx_memory_photos_object_key (object_key)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='时光相册图片';

CREATE TABLE IF NOT EXISTS task_ai_summaries (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '总结ID',
    summary_date DATE NOT NULL COMMENT '总结日期（Asia/Shanghai）',
    facts_hash CHAR(16) NOT NULL COMMENT '任务事实摘要，用于判断是否需要重新生成',
    payload_json JSON NOT NULL COMMENT 'TaskAISummaryResponse JSON',
    ai_generated TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否由AI生成，失败时保存规则总结',
    generated_at DATETIME(3) NOT NULL COMMENT '实际生成时间',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_task_ai_summaries_date (summary_date),
    KEY idx_task_ai_summaries_generated_at (generated_at)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='每日任务AI总结';

CREATE TABLE IF NOT EXISTS daily_routines (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '每日事项ID',
    title VARCHAR(120) NOT NULL COMMENT '事项名称',
    time_of_day CHAR(5) NOT NULL COMMENT '每日执行时间HH:mm',
    repeat_unit VARCHAR(16) NOT NULL DEFAULT 'day' COMMENT '重复单位：day/week/month',
    repeat_interval INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '重复间隔',
    start_date DATE NULL COMMENT '重复周期起始日期',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '所属用户ID，逻辑关联users.id',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at DATETIME(3) NULL COMMENT '软删除时间',
    PRIMARY KEY (id),
    KEY idx_daily_routines_user (created_by, deleted_at, enabled, time_of_day)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='个人每日事项模板';

CREATE TABLE IF NOT EXISTS daily_checkins (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '每日事项打卡ID',
    routine_id BIGINT UNSIGNED NOT NULL COMMENT '每日事项ID，逻辑关联daily_routines.id',
    checkin_date DATE NOT NULL COMMENT '打卡日期',
    completed TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否完成',
    checked_at DATETIME(3) NULL COMMENT '实际完成时间',
    created_by BIGINT UNSIGNED NOT NULL COMMENT '所属用户ID，逻辑关联users.id',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_daily_checkins_user_routine_date (created_by, routine_id, checkin_date),
    KEY idx_daily_checkins_user_date (created_by, checkin_date)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='个人每日事项打卡记录';

CREATE TABLE IF NOT EXISTS user_module_permissions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '权限ID',
    user_id BIGINT UNSIGNED NOT NULL COMMENT '用户ID，逻辑关联users.id',
    module VARCHAR(32) NOT NULL COMMENT 'task/activity_log/bill/restaurant/place',
    permission_level VARCHAR(16) NOT NULL DEFAULT 'none' COMMENT 'none/read/write',
    granted_by BIGINT UNSIGNED NOT NULL COMMENT '授权管理员ID，逻辑关联users.id',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_module_permissions_user_module (user_id, module),
    KEY idx_user_module_permissions_granted_by (granted_by, updated_at)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COLLATE=utf8mb4_general_ci
  COMMENT='用户模块权限';

INSERT IGNORE INTO user_module_permissions (user_id, module, permission_level, granted_by)
SELECT users.id, modules.module, 'write', users.id
FROM users
JOIN (
    SELECT 'task' AS module
    UNION ALL SELECT 'activity_log'
    UNION ALL SELECT 'bill'
    UNION ALL SELECT 'restaurant'
    UNION ALL SELECT 'place'
) AS modules
WHERE users.status = 'approved';
