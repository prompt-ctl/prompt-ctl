package prompt

import "strings"

// DomainKnowledge represents expertise knowledge for a specific domain
type DomainKnowledge struct {
	ExpertRole     string
	KeyConcerns    []string
	OutputSections []string
	Constraints    []string
}

// domainKnowledgeMap stores expert knowledge for each domain
var domainKnowledgeMap = map[string]*DomainKnowledge{
	"gaming": {
		ExpertRole: "senior game developer and game designer with 10+ years shipping titles, deep experience in game architecture, player psychology, and engagement mechanics",
		KeyConcerns: []string{
			"Core game loop and what makes it addictive",
			"Data model for game state, player profiles, and progression",
			"Real-time vs turn-based mechanics and their technical implications",
			"Player engagement: retention hooks, daily rewards, progression curves",
			"Multiplayer: matchmaking, leaderboards, social features",
			"Monetization: free-to-play vs premium, in-app purchases, fairness",
			"Tech stack: backend for game state, real-time sync, offline support",
			"MVP scope: what to build first vs what to defer",
			"Content pipeline: how to create and balance game content at scale",
		},
		OutputSections: []string{
			"GAME CONCEPT: Core loop, unique hook, target audience",
			"ARCHITECTURE: System design, data model, state management",
			"GAME MECHANICS: Rules, progression, balance considerations",
			"TECH STACK: Recommended technologies with trade-offs",
			"MVP DEFINITION: Phase 1 scope with exact feature list",
			"BUILD SEQUENCE: Week-by-week development order",
			"ENGAGEMENT DESIGN: Retention mechanics, session design",
			"MONETIZATION: Revenue model options with pros/cons",
		},
		Constraints: []string{
			"Prioritize the core game loop - if it's not fun in a text description, it won't be fun in code",
			"Specify exact data models with field names, not vague descriptions",
			"For each feature, state whether it's MVP or post-launch",
			"Include at least one non-obvious technical challenge and how to handle it",
			"Recommend specific libraries/frameworks, not generic categories",
		},
	},
	"fintech": {
		ExpertRole: "senior fintech architect with experience building payment systems, trading platforms, and banking applications, deep knowledge of financial regulations and security requirements",
		KeyConcerns: []string{
			"Regulatory compliance (PSD2, PCI-DSS, KYC/AML, SOC2)",
			"Transaction integrity: ACID properties, idempotency, reconciliation",
			"Security: encryption at rest/transit, key management, audit trails",
			"Money handling: decimal precision, currency conversion, rounding",
			"Integration patterns: banking APIs, payment processors, market data",
			"Scalability: high-throughput transaction processing",
			"Error handling: partial failures, compensating transactions, rollbacks",
		},
		OutputSections: []string{
			"SYSTEM OVERVIEW: What it does, who uses it, money flow",
			"ARCHITECTURE: Services, data stores, message flows",
			"DATA MODEL: Accounts, transactions, ledger entries with exact schemas",
			"SECURITY & COMPLIANCE: Requirements and implementation approach",
			"INTEGRATION DESIGN: External services, API contracts, failure modes",
			"MVP SCOPE: Phase 1 features with regulatory minimum",
			"BUILD SEQUENCE: Implementation order respecting dependencies",
			"RISK REGISTER: Technical and regulatory risks with mitigations",
		},
		Constraints: []string{
			"Never suggest storing financial data without specifying encryption approach",
			"All money amounts must use decimal/BigDecimal, never floating point",
			"Specify exact compliance requirements for the described jurisdiction",
			"Include idempotency strategy for every write operation",
			"Error scenarios must be explicitly addressed, not hand-waved",
		},
	},
	"saas": {
		ExpertRole: "senior SaaS architect and product engineer who has built and scaled B2B platforms from 0 to 10k customers, experienced with multi-tenancy, billing systems, and enterprise readiness",
		KeyConcerns: []string{
			"Multi-tenancy architecture: data isolation, shared vs dedicated",
			"Authentication & authorization: SSO, RBAC, team management",
			"Billing & subscriptions: metering, tiers, usage tracking, invoicing",
			"Onboarding: time-to-value, setup wizard, sample data",
			"API design: versioning, rate limiting, webhook events",
			"Scalability: per-tenant resource isolation, noisy neighbor prevention",
			"Enterprise readiness: audit logs, data export, custom domains",
		},
		OutputSections: []string{
			"PRODUCT DEFINITION: Core value prop, target user, workflow",
			"ARCHITECTURE: Multi-tenant design, service boundaries, data model",
			"AUTH & TEAMS: User model, roles, permissions, invite flow",
			"CORE FEATURES: Detailed feature specs for MVP",
			"BILLING: Tier structure, metering approach, payment integration",
			"API DESIGN: Key endpoints, contracts, webhook events",
			"TECH STACK: Recommended stack with reasoning",
			"MVP SCOPE & BUILD ORDER: What to ship first and why",
		},
		Constraints: []string{
			"Specify multi-tenancy strategy explicitly (row-level, schema-level, or database-level)",
			"Include at least 3 pricing tiers with specific feature gates",
			"Address data isolation and what happens if a tenant is deleted",
			"Define the onboarding flow step by step",
			"Recommend specific services/libraries, not generic categories",
		},
	},
	"ecommerce": {
		ExpertRole: "senior e-commerce engineer with experience building marketplaces and shopping platforms, deep knowledge of payment processing, inventory management, and conversion optimization",
		KeyConcerns: []string{
			"Product catalog: categories, variants, attributes, search/filter",
			"Cart & checkout: multi-step, guest checkout, abandoned cart recovery",
			"Payments: processor integration, refunds, disputes, multi-currency",
			"Inventory: stock tracking, reservations, backorders, warehousing",
			"Order lifecycle: placement, fulfillment, shipping, returns",
			"Search & discovery: full-text search, faceted filtering, recommendations",
			"Performance: page load speed, image optimization, caching",
		},
		OutputSections: []string{
			"PLATFORM OVERVIEW: Market positioning, seller/buyer model, differentiator",
			"PRODUCT & CATALOG: Data model, categories, search architecture",
			"CART & CHECKOUT: Flow design, payment integration, conversion optimization",
			"ORDER MANAGEMENT: Lifecycle, fulfillment, shipping, returns",
			"USER SYSTEM: Buyer/seller profiles, reviews, trust mechanics",
			"TECH STACK: Frontend, backend, search, CDN, payments",
			"MVP SCOPE: Launch features vs phase 2",
			"GROWTH MECHANICS: SEO, referrals, retention hooks",
		},
		Constraints: []string{
			"Specify exact payment processor and integration approach",
			"Address inventory consistency under concurrent purchases",
			"Include mobile-first design considerations",
			"Define the search ranking algorithm approach",
			"Include specific conversion optimization tactics in the checkout flow",
		},
	},
	"mobile": {
		ExpertRole: "senior mobile engineer with experience shipping iOS and Android apps, deep knowledge of mobile-specific challenges like offline support, push notifications, and app store optimization",
		KeyConcerns: []string{
			"Platform strategy: native vs cross-platform (React Native, Flutter)",
			"Offline-first architecture: local storage, sync conflicts, queue management",
			"Push notifications: segmentation, deep linking, opt-in rates",
			"Performance: app size, startup time, memory usage, battery drain",
			"App store presence: ASO, screenshots, ratings strategy",
			"Mobile UX patterns: gestures, navigation, responsive layouts",
			"Device capabilities: camera, location, biometrics, background tasks",
		},
		OutputSections: []string{
			"APP CONCEPT: Core user flow, platform choice justification",
			"ARCHITECTURE: Data sync strategy, state management, API design",
			"OFFLINE SUPPORT: Conflict resolution, queue management, storage",
			"PUSH & ENGAGEMENT: Notification strategy, deep linking, retention",
			"UX DESIGN: Navigation, key screens, mobile-specific patterns",
			"TECH STACK: Framework choice, key libraries, backend requirements",
			"MVP SCOPE: Phase 1 features and platform priorities",
			"APP STORE STRATEGY: Launch plan, ASO, user acquisition",
		},
		Constraints: []string{
			"Justify native vs cross-platform with specific trade-offs for this use case",
			"Specify offline data strategy with conflict resolution approach",
			"Include app size budget and performance targets",
			"Address platform-specific differences (iOS vs Android)",
			"Include push notification strategy with opt-in rate considerations",
		},
	},
	"ai_ml": {
		ExpertRole: "senior ML engineer with experience deploying production ML systems, deep knowledge of model selection, training pipelines, and production ML infrastructure",
		KeyConcerns: []string{
			"Problem formulation: classification, regression, generation, or retrieval",
			"Data pipeline: collection, labeling, validation, versioning",
			"Model selection: trade-offs between accuracy, latency, cost, interpretability",
			"Training infrastructure: compute, experiment tracking, hyperparameter tuning",
			"Evaluation: metrics, test sets, A/B testing, monitoring drift",
			"Production deployment: serving, scaling, monitoring, retraining triggers",
			"ML-specific risks: bias, fairness, privacy, adversarial robustness",
		},
		OutputSections: []string{
			"PROBLEM DEFINITION: ML task type, success metrics, baseline",
			"DATA STRATEGY: Sources, labeling approach, size requirements, versioning",
			"MODEL APPROACH: Architecture options, trade-off analysis, recommendation",
			"TRAINING PIPELINE: Infrastructure, experiment tracking, validation strategy",
			"EVALUATION: Metrics, test methodology, performance targets",
			"DEPLOYMENT: Serving architecture, latency/cost budget, monitoring",
			"MVP SCOPE: Simplest viable model vs full solution",
			"RISK MITIGATION: Bias, drift, failure modes, safeguards",
		},
		Constraints: []string{
			"Specify exact metrics for success (not just 'high accuracy')",
			"Include data size requirements and labeling strategy",
			"Address the cold-start problem if applicable",
			"Include monitoring for model drift and retraining triggers",
			"Recommend specific model architectures/libraries with reasoning",
		},
	},
	"devops": {
		ExpertRole: "senior DevOps/SRE engineer with experience building CI/CD pipelines, container orchestration, and production reliability at scale",
		KeyConcerns: []string{
			"CI/CD pipeline: build, test, deploy automation, rollback strategy",
			"Infrastructure as code: Terraform, CloudFormation, provisioning",
			"Container orchestration: Kubernetes, Docker, service mesh",
			"Monitoring & observability: metrics, logs, traces, alerting",
			"Reliability: SLOs, error budgets, incident response, post-mortems",
			"Security: secrets management, network policies, vulnerability scanning",
			"Cost optimization: resource allocation, autoscaling, waste reduction",
		},
		OutputSections: []string{
			"SYSTEM OVERVIEW: Current state, target state, migration path",
			"CI/CD DESIGN: Pipeline stages, testing strategy, deployment approach",
			"INFRASTRUCTURE: IaC approach, provider choice, network architecture",
			"ORCHESTRATION: Container strategy, scaling, service discovery",
			"OBSERVABILITY: Monitoring stack, key metrics, alerting rules",
			"RELIABILITY: SLOs, error budget, incident runbooks",
			"SECURITY: Secrets, scanning, policies, compliance",
			"IMPLEMENTATION PLAN: Phased rollout, risks, rollback plan",
		},
		Constraints: []string{
			"Specify exact tools/platforms with version constraints",
			"Include disaster recovery and backup strategy",
			"Define SLOs with specific percentiles and time windows",
			"Address secrets management explicitly",
			"Include cost estimates or optimization strategy",
		},
	},
	"education": {
		ExpertRole: "senior education technology specialist with experience building learning platforms, deep knowledge of pedagogy, learner engagement, and assessment design",
		KeyConcerns: []string{
			"Learning objectives: Bloom's taxonomy, skill progression, outcomes",
			"Content structure: courses, modules, lessons, adaptive paths",
			"Assessment design: formative vs summative, auto-grading, rubrics",
			"Engagement mechanics: gamification, progress tracking, social learning",
			"Accessibility: WCAG compliance, screen readers, diverse learners",
			"Analytics: completion rates, time-on-task, learning gains, dropout prediction",
			"Content delivery: video, interactive exercises, quizzes, projects",
		},
		OutputSections: []string{
			"LEARNING GOALS: Target audience, skill outcomes, success criteria",
			"COURSE STRUCTURE: Module breakdown, learning path, pacing",
			"CONTENT DESIGN: Media types, interactive elements, assessment strategy",
			"ENGAGEMENT: Motivation hooks, progress mechanics, social features",
			"PLATFORM ARCHITECTURE: Content management, user system, analytics",
			"TECH STACK: LMS vs custom, video hosting, assessment tools",
			"MVP SCOPE: Core learning loop vs advanced features",
			"MEASUREMENT: Analytics, learner feedback, improvement cycles",
		},
		Constraints: []string{
			"Define specific learning objectives with measurable outcomes",
			"Include accessibility considerations (WCAG 2.1 AA minimum)",
			"Specify assessment strategy with grading/feedback approach",
			"Address different learning styles and pacing needs",
			"Include engagement metrics to track",
		},
	},
	"healthcare": {
		ExpertRole: "senior healthcare IT architect with experience building HIPAA-compliant systems, deep knowledge of clinical workflows, medical data standards, and regulatory requirements",
		KeyConcerns: []string{
			"Regulatory compliance: HIPAA, GDPR, FDA (if medical device), state laws",
			"Data security: PHI encryption, access controls, audit trails, BAAs",
			"Clinical workflows: provider experience, patient experience, alerts/notifications",
			"Interoperability: HL7, FHIR, EHR integration, data exchange",
			"Medical accuracy: validation, clinical decision support, error prevention",
			"Reliability: uptime requirements, disaster recovery, data backup",
			"User roles: patients, providers, admins, billing, different permission levels",
		},
		OutputSections: []string{
			"CLINICAL USE CASE: Problem being solved, user workflows, care impact",
			"COMPLIANCE STRATEGY: HIPAA requirements, BAA needs, audit approach",
			"ARCHITECTURE: Services, data stores, encryption, access controls",
			"DATA MODEL: Patient records, clinical data, standards (FHIR/HL7)",
			"INTEGRATION: EHR connections, data exchange, API design",
			"CLINICAL WORKFLOW: Provider steps, patient steps, notifications",
			"TECH STACK: HIPAA-compliant hosting, frameworks, services",
			"RISK MITIGATION: Security, compliance, clinical safety, validation",
		},
		Constraints: []string{
			"Specify HIPAA compliance approach for all PHI storage and transmission",
			"Include BAA (Business Associate Agreement) requirements with vendors",
			"Address audit logging with specific retention requirements",
			"Specify data encryption at rest and in transit",
			"Include clinical validation and error prevention mechanisms",
		},
	},
	"social": {
		ExpertRole: "senior social platform engineer with experience building community platforms, deep knowledge of content moderation, viral mechanics, and social graph architecture",
		KeyConcerns: []string{
			"Social graph: follow model, friend model, privacy controls",
			"Feed algorithm: chronological vs algorithmic, ranking signals, personalization",
			"Content moderation: automated detection, reporting, human review, appeals",
			"Viral mechanics: sharing, notifications, discovery, network effects",
			"Privacy & safety: blocking, muting, data control, minor protection",
			"Scalability: hot users, viral content, read-heavy optimization",
			"Engagement: retention loops, notification strategy, FOMO mechanics",
		},
		OutputSections: []string{
			"PLATFORM CONCEPT: Core interaction, unique value vs existing platforms",
			"SOCIAL GRAPH: Relationship model, privacy tiers, discovery",
			"CONTENT SYSTEM: Post types, media handling, editing/deletion",
			"FEED & DISCOVERY: Algorithm approach, personalization, trending",
			"MODERATION: Detection, reporting, review process, transparency",
			"ENGAGEMENT: Notifications, viral mechanics, retention hooks",
			"TECH STACK: Backend, CDN, media storage, recommendation engine",
			"GROWTH & SAFETY: Acquisition, retention, abuse prevention",
		},
		Constraints: []string{
			"Specify social graph model (symmetric vs asymmetric follows)",
			"Include content moderation strategy with automation and human review",
			"Address privacy controls and user data export",
			"Define notification strategy with frequency caps",
			"Include abuse prevention and safety features",
		},
	},
	"general": {
		ExpertRole: "senior software architect and product strategist with broad experience across web applications, APIs, and system design",
		KeyConcerns: []string{
			"Core problem being solved and for whom",
			"System architecture and key technical decisions",
			"Data model and storage strategy",
			"User experience flow and key interactions",
			"Tech stack selection with trade-offs",
			"MVP scope vs full vision",
			"Scalability considerations",
			"Security and data privacy",
		},
		OutputSections: []string{
			"CONCEPT: What it does, who it's for, why it matters",
			"ARCHITECTURE: System design, components, data flow",
			"DATA MODEL: Core entities, relationships, schemas",
			"KEY FEATURES: Detailed specs for each core feature",
			"TECH STACK: Recommended technologies with reasoning",
			"MVP DEFINITION: Phase 1 scope with build order",
			"TECHNICAL CHALLENGES: Non-obvious problems and solutions",
			"NEXT STEPS: Immediate action items to start building",
		},
		Constraints: []string{
			"Be specific - name exact technologies, not categories",
			"Include data model with actual field names and types",
			"Separate MVP features from nice-to-haves explicitly",
			"Address at least 2 non-obvious technical challenges",
			"End with concrete, actionable next steps (not vague advice)",
		},
	},
}

// detectDomain analyzes intent and returns the best matching domain
// Ordered by specificity - more specific domains first to avoid false positives
func detectDomain(lower string) string {
	domainPatterns := []struct {
		keywords []string
		domain   string
	}{
		// Healthcare - check first due to specific medical terminology
		{[]string{
			"patient", "medical", "health record", "diagnosis", "hipaa", "clinical", "healthcare", "hospital", 
			"doctor", "clinic", "prescription", "telemedicine", "telehealth", "ehr", "emr", "radiology",
			"pharmacy", "nursing", "surgery", "physician", "therapist", "dental", "veterinary", "wellness",
			"mental health", "psychiatry", "cardiology", "oncology", "lab results", "medical imaging",
			"electronic health", "medical device", "healthcare provider", "medical records", "patient portal",
			"appointment scheduling", "medical billing", "insurance claim", "icd-10", "cpt code", "medication",
			"vaccine", "immunization", "vital signs", "blood pressure", "diagnostic", "symptom", "epidemic",
		}, "healthcare"},
		
		// Fintech - specific financial terms
		{[]string{
			"payment", "trading", "banking", "transaction", "ledger", "invoice", "fintech", "wallet", 
			"currency", "financial", "cryptocurrency", "blockchain", "exchange", "bitcoin", "ethereum",
			"defi", "nft", "token", "payment gateway", "payment processor", "stripe", "paypal", "square",
			"merchant", "pos", "point of sale", "credit card", "debit card", "wire transfer", "ach",
			"bank account", "checking", "savings", "investment", "portfolio", "stock", "bond", "forex",
			"remittance", "cross-border payment", "money transfer", "escrow", "kyc", "aml", "compliance",
			"fraud detection", "risk management", "underwriting", "loan", "mortgage", "credit score",
			"accounting", "bookkeeping", "reconciliation", "settlement", "clearing house", "swift",
			"crypto exchange", "trading platform", "algorithmic trading", "market data", "financial data",
		}, "fintech"},
		
		// Gaming - specific game terms
		{[]string{
			"game", "player", "rpg", "multiplayer", "score", "level", "match", "tournament", "leaderboard", 
			"achievement", "football", "soccer", "manager", "fps", "mmorpg", "moba", "rts game", "simulation",
			"strategy game", "puzzle game", "casual game", "indie game", "aaa game", "game engine", "unity3d",
			"unreal engine", "godot", "game design", "game mechanics", "gameplay", "game loop", "npc", "ai opponent",
			"character", "avatar", "inventory", "quest", "mission", "campaign", "dungeon", "raid", "guild",
			"clan", "pvp", "pve", "co-op", "battle royale", "esports", "streaming", "twitch", "discord",
			"game server", "matchmaking", "anti-cheat", "game balance", "progression system", "skill tree",
			"loot box", "microtransaction", "season pass", "dlc", "game monetization", "f2p", "gacha",
			"sports game", "racing game", "fighting game", "platformer", "roguelike", "metroidvania",
		}, "gaming"},
		
		// E-commerce - specific shopping terms (but not generic "listing")
		{[]string{
			"shop", "cart", "product catalog", "marketplace", "checkout", "ecommerce", "e-commerce", 
			"inventory", "shopify", "woocommerce", "stripe payment", "magento", "bigcommerce", "amazon",
			"ebay", "etsy", "online store", "webshop", "shopping cart", "order", "purchase", "buy",
			"sell", "vendor", "merchant", "sku", "product variant", "stock", "warehouse", "fulfillment",
			"shipping", "delivery", "tracking", "return", "refund", "customer review", "rating",
			"wishlist", "comparison", "search filter", "faceted search", "product recommendation",
			"cross-sell", "upsell", "discount", "coupon", "promo code", "sale", "pricing", "payment method",
			"order management", "inventory management", "dropshipping", "wholesale", "retail", "b2c",
			"storefront", "product page", "category page", "product image", "product description",
			"shopping experience", "conversion rate", "abandoned cart", "checkout flow", "one-click buy",
		}, "ecommerce"},
		
		// Mobile - iOS, Android, mobile-specific
		{[]string{
			"ios", "android", "mobile", "smartphone", "tablet", "app store", "play store", "react native", 
			"flutter", "swift", "kotlin", "objective-c", "java android", "xamarin", "cordova", "phonegap", "ionic",
			"mobile app", "native app", "hybrid app", "cross-platform", "mobile development", "mobile ui",
			"mobile ux", "touch gesture", "swipe", "tap", "push notification", "local notification",
			"deep linking", "universal link", "app link", "in-app purchase", "mobile payment", "apple pay",
			"google pay", "mobile wallet", "offline mode", "offline sync", "background sync", "geolocation",
			"gps", "camera", "photo", "video", "augmented reality", "arkit", "arcore", "core ml", "ml kit", "biometric",
			"face id", "touch id", "fingerprint", "mobile security", "app signing", "provisioning profile",
			"testflight", "app distribution", "mobile analytics", "crash reporting", "firebase", "realm",
			"mobile database", "sqlite", "core data", "app performance", "battery optimization", "memory",
		}, "mobile"},
		
		// AI/ML - machine learning specific (check before general "train")
		{[]string{
			"ml model", "neural", "dataset", "predict", "machine learning", "deep learning", "classification", 
			"regression", "tensorflow", "pytorch", "recommendation system", "sentiment analysis", "nlp",
			"natural language processing", "computer vision", "image recognition", "object detection",
			"neural network", "cnn", "rnn", "lstm", "gru", "transformer", "bert", "gpt", "llm",
			"large language model", "generative ai", "diffusion model", "gan", "autoencoder", "reinforcement",
			"supervised learning", "unsupervised learning", "semi-supervised", "transfer learning",
			"fine-tuning", "model training", "hyperparameter", "feature engineering", "data preprocessing",
			"model evaluation", "cross-validation", "overfitting", "underfitting", "regularization",
			"gradient descent", "backpropagation", "activation function", "loss function", "optimizer",
			"scikit-learn", "keras", "hugging face", "openai", "anthropic", "langchain", "vector database",
			"embedding", "tokenization", "model deployment", "model serving", "mlops", "model monitoring",
			"data labeling", "annotation", "training data", "test data", "validation set", "ai ethics",
		}, "ai_ml"},
		
		// DevOps - infrastructure and deployment (before general "cluster")
		{[]string{
			"ci/cd", "pipeline", "kubernetes", "docker", "devops", "infrastructure", "terraform", "ansible", 
			"jenkins", "helm", "deploy", "deployment", "k8s", "container", "containerization", "orchestration",
			"gitlab ci", "github actions", "circleci", "travis ci", "build automation", "continuous integration",
			"continuous deployment", "continuous delivery", "release management", "version control", "git",
			"gitops", "infrastructure as code", "iac", "cloudformation", "pulumi", "vagrant", "packer",
			"monitoring", "observability", "prometheus", "grafana", "datadog", "new relic", "logging",
			"elk stack", "splunk", "alerting", "incident response", "on-call", "sre", "site reliability",
			"aws", "azure", "gcp", "cloud", "serverless", "lambda", "cloud function", "vpc", "networking",
			"load balancer", "autoscaling", "horizontal scaling", "vertical scaling", "blue-green deployment",
			"canary deployment", "rolling update", "service mesh", "istio", "consul", "vault", "secrets",
			"configuration management", "docker compose", "dockerfile", "registry", "artifact repository",
		}, "devops"},
		
		// SaaS - multi-tenant, subscription-based (check before general analytics)
		{[]string{
			"saas", "subscription", "tenant", "multi-tenant", "b2b", "b2c", "billing", "usage tracking", 
			"api key", "customer churn", "retention", "mrr", "arr", "recurring revenue", "pricing tier",
			"freemium", "trial", "seat", "per-user pricing", "usage-based", "metered billing", "invoice",
			"payment processing", "subscription management", "customer lifecycle", "onboarding", "activation",
			"engagement", "product-led growth", "self-service", "admin dashboard", "user management",
			"role-based access", "rbac", "sso", "single sign-on", "oauth", "saml", "api management",
			"webhook", "rate limiting", "quota", "feature flag", "a/b testing", "analytics", "metrics",
			"customer success", "support ticket", "help desk", "knowledge base", "documentation portal",
			"white label", "custom domain", "branding", "workspace", "organization", "team collaboration",
			"data export", "api integration", "third-party integration", "marketplace", "app directory",
			"cohort analysis", "funnel analysis", "conversion tracking", "retention analysis",
		}, "saas"},
		
		// Education - learning platforms
		{[]string{
			"course", "quiz", "student", "curriculum", "lms", "education", "teach", "lesson", "assignment", 
			"grade", "learning platform", "e-learning", "online learning", "mooc", "virtual classroom",
			"edtech", "educational technology", "learning management", "student information system", "sis",
			"gradebook", "attendance", "enrollment", "class", "instructor", "teacher", "professor", "tutor",
			"tutoring", "homework", "exam", "test", "assessment", "rubric", "learning objective", "syllabus",
			"course content", "video lecture", "interactive lesson", "educational game", "gamification",
			"badge", "certificate", "credential", "accreditation", "degree", "diploma", "transcript",
			"academic", "school", "university", "college", "k-12", "higher education", "vocational",
			"training program", "workshop", "webinar", "cohort", "peer review", "discussion forum",
			"learning path", "competency", "skill development", "personalized learning", "adaptive learning",
		}, "education"},
		
		// Social - social networking features (avoid conflicts with "post" in other contexts)
		{[]string{
			"feed", "profile", "follower", "following", "messaging", "social network", "friend", "like", 
			"comment", "share", "social media", "timeline", "newsfeed", "activity stream", "notification",
			"mention", "tag", "hashtag", "trending", "viral", "engagement", "reaction", "emoji", "story",
			"status update", "live stream", "video sharing", "photo sharing", "album", "group", "community",
			"chat", "direct message", "dm", "conversation", "thread", "reply", "retweet", "repost",
			"user profile", "bio", "avatar", "cover photo", "privacy setting", "block", "mute", "report",
			"moderation", "content policy", "community guidelines", "user-generated content", "ugc",
			"social graph", "connection", "network", "relationship", "mutual friend", "suggestion",
			"people you may know", "discover", "explore", "influencer", "verified", "social login",
			"social sharing", "social integration", "social widget", "social plugin", "embed", "share button",
		}, "social"},
		
		// Analytics/Dashboard (maps to SaaS for business intelligence)
		{[]string{
			"dashboard", "analytics", "metrics", "chart", "visualization", "reporting", "kpi", "user behavior",
			"business intelligence", "bi", "data analytics", "data visualization", "tableau", "power bi",
			"looker", "metabase", "redash", "superset", "data warehouse", "etl", "data pipeline", "olap",
			"data aggregation", "time series", "cohort analysis", "funnel analysis", "retention analysis",
			"conversion tracking", "event tracking", "user tracking", "session replay", "heatmap", "clickstream",
			"a/b test results", "experiment", "statistical significance", "segment", "audience", "attribution",
			"google analytics", "mixpanel", "amplitude", "heap", "segment", "real-time analytics", "historical data",
			"drill down", "filter", "dimension", "measure", "calculated field", "custom metric", "benchmark",
		}, "saas"},
	}

	for _, p := range domainPatterns {
		for _, kw := range p.keywords {
			if strings.Contains(lower, kw) {
				return p.domain
			}
		}
	}

	return "general"
}
