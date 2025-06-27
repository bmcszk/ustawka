# Task Tracking: Enhanced Act Lifecycle Tracking System

**PRD Reference**: [Enhanced Act Lifecycle Tracking](./PRD-Enhanced-Act-Lifecycle-Tracking.md)  
**Created**: 2025-06-27  
**Status**: Ready for Implementation  

## Epic Overview

Transform Ustawka from basic Act monitoring to comprehensive Polish legislative transparency platform with real-time tracking, voting analysis, and enhanced UX.

**Target Completion**: 10 sprints (~20 weeks)  
**Success Criteria**: 90%+ lifecycle coverage, 95%+ voting data completeness, 50% engagement increase

---

## Phase 1: Data Infrastructure 🏗️
**Timeline**: Sprint 1-2 (Weeks 1-4)  
**Goal**: Enhanced data foundation with multi-source integration

### Sprint 1: Database Schema & Core APIs

#### ✅ **Task 1.1**: Enhanced Database Schema
- [ ] **Subtask 1.1.1**: Design enhanced `acts` table schema
  - Add columns: `detailed_status`, `current_stage`, `stage_date`, `days_in_stage`
  - Add columns: `initiator_type`, `committee_code`, `rapporteur_name`
  - Add columns: `urgency_status`, `eu_compliance`, `process_print_number`, `rcl_link`
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: None

- [ ] **Subtask 1.1.2**: Create `act_votes` table
  - Define schema for Sejm/Senate voting records
  - Include JSONB field for full voting details
  - Set up proper indexing for performance
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: 1.1.1

- [ ] **Subtask 1.1.3**: Create `party_votes` table
  - Schema for party-level voting breakdowns
  - Link to act_votes with foreign keys
  - Include vote counts by party
  - **Estimate**: 0.5 days
  - **Owner**: Backend Developer
  - **Dependencies**: 1.1.2

- [ ] **Subtask 1.1.4**: Create `act_stages` table
  - Track process stages with timestamps
  - Include committee assignments and notes
  - Set up stage progression tracking
  - **Estimate**: 0.5 days
  - **Owner**: Backend Developer
  - **Dependencies**: 1.1.1

- [ ] **Subtask 1.1.5**: Database migration scripts
  - Create migration for existing data
  - Test migration on development data
  - Plan production migration strategy
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: 1.1.1-1.1.4

#### ✅ **Task 1.2**: Enhanced Sejm API Integration
- [ ] **Subtask 1.2.1**: Extend Sejm client for process tracking
  - Implement `/sejm/term{X}/processes/{number}` endpoint
  - Parse stage progression data
  - Extract committee and rapporteur information
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: None

- [ ] **Subtask 1.2.2**: Implement voting data fetching
  - Add `/sejm/term{X}/votings` endpoint integration
  - Parse individual MP votes and party breakdowns
  - Handle vote linking to Acts
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: 1.2.1

- [ ] **Subtask 1.2.3**: Enhanced data models
  - Update Go structs for `EnhancedAct`
  - Create `VotingRecord` and `PartyVote` types
  - Implement `ProcessStage` tracking
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: 1.1.1-1.1.4

#### ✅ **Task 1.3**: Basic Status Enrichment Logic
- [ ] **Subtask 1.3.1**: Status determination algorithm
  - Map Sejm process stages to enhanced statuses
  - Handle edge cases and missing data
  - Implement status progression rules
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: 1.2.1

- [ ] **Subtask 1.3.2**: Days in stage calculation
  - Calculate time spent in current stage
  - Handle weekends and holidays
  - Add performance indicators
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: 1.3.1

### Sprint 2: Senate Integration & Data Pipeline

#### ✅ **Task 2.1**: Senate Data Integration
- [ ] **Subtask 2.1.1**: Senate open data client
  - Parse XML manifest from Senate API
  - Download and process CSV voting files
  - Handle both individual and party voting data
  - **Estimate**: 3 days
  - **Owner**: Backend Developer
  - **Dependencies**: 1.1.2, 1.1.3

- [ ] **Subtask 2.1.2**: Senate-Sejm Act linking
  - Implement title-based matching algorithm
  - Handle timing-based correlation
  - Manual override capability for edge cases
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: 2.1.1

#### ✅ **Task 2.2**: Data Pipeline Architecture
- [ ] **Subtask 2.2.1**: Polling system for Sejm API
  - Implement 30-minute polling cycle
  - Handle rate limiting and errors gracefully
  - Log all API interactions
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: 1.2.1, 1.2.2

- [ ] **Subtask 2.2.2**: Daily Senate data updates
  - Schedule daily Senate data fetching
  - Process new voting files incrementally
  - Update existing records when needed
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: 2.1.1

- [ ] **Subtask 2.2.3**: Data quality monitoring
  - Implement data validation rules
  - Alert system for missing or inconsistent data
  - Metrics dashboard for data completeness
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: 2.2.1, 2.2.2

---

## Phase 2: Enhanced Board Interface 🎨
**Timeline**: Sprint 3-4 (Weeks 5-8)  
**Goal**: New multi-column board with rich Act information

### Sprint 3: New Board Layout & Status Columns

#### ✅ **Task 3.1**: Multi-Column Board Component
- [ ] **Subtask 3.1.1**: Enhanced board layout design
  - Design 5-7 status column layout
  - Implement horizontal scrolling
  - Responsive design for different screen sizes
  - **Estimate**: 2 days
  - **Owner**: Frontend Developer
  - **Dependencies**: None

- [ ] **Subtask 3.1.2**: Status column configuration
  - Define status groupings for columns
  - Implement column titles and colors
  - Add column count indicators
  - **Estimate**: 1 day
  - **Owner**: Frontend Developer
  - **Dependencies**: 3.1.1

- [ ] **Subtask 3.1.3**: Column state management
  - Implement state for multiple columns
  - Handle Act movement between columns
  - Add drag-and-drop capability (future)
  - **Estimate**: 2 days
  - **Owner**: Frontend Developer
  - **Dependencies**: 3.1.2

#### ✅ **Task 3.2**: Enhanced Act Card Design
- [ ] **Subtask 3.2.1**: Rich information card layout
  - Design card with 10+ data points
  - Implement progressive disclosure
  - Add visual indicators for urgency/status
  - **Estimate**: 3 days
  - **Owner**: Frontend Developer
  - **Dependencies**: None

- [ ] **Subtask 3.2.2**: Voting information preview
  - Show party voting breakdowns on card
  - Quick vote result indicators
  - Click-through to detailed voting page
  - **Estimate**: 2 days
  - **Owner**: Frontend Developer
  - **Dependencies**: 3.2.1

#### ✅ **Task 3.3**: Backend API Updates
- [ ] **Subtask 3.3.1**: Enhanced Acts endpoint
  - Update `/acts/{year}` to return enhanced data
  - Group by detailed status for columns
  - Add voting summary information
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: 1.3.1

- [ ] **Subtask 3.3.2**: Column statistics endpoint
  - Create `/acts/{year}/columns` endpoint
  - Return counts per status column
  - Include performance metrics
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: 3.3.1

### Sprint 4: Enhanced Filtering & Search

#### ✅ **Task 4.1**: Advanced Filtering System
- [ ] **Subtask 4.1.1**: Multi-criteria filter interface
  - Status, year, initiator, committee filters
  - Timeline and voting stage filters
  - Tag and type filtering
  - **Estimate**: 3 days
  - **Owner**: Frontend Developer
  - **Dependencies**: 3.1.1

- [ ] **Subtask 4.1.2**: Filter state management
  - Persistent filter state in URL
  - Save user filter preferences
  - Quick filter presets
  - **Estimate**: 2 days
  - **Owner**: Frontend Developer
  - **Dependencies**: 4.1.1

#### ✅ **Task 4.2**: Enhanced Search Functionality
- [ ] **Subtask 4.2.1**: Full-text search implementation
  - Search in titles, descriptions, ELI IDs
  - Add search result highlighting
  - Implement search suggestions
  - **Estimate**: 2 days
  - **Owner**: Full-stack Developer
  - **Dependencies**: None

- [ ] **Subtask 4.2.2**: Advanced search filters
  - MP/Senator name search for voting
  - Committee and rapporteur search
  - Date range search capabilities
  - **Estimate**: 1 day
  - **Owner**: Full-stack Developer
  - **Dependencies**: 4.2.1

---

## Phase 3: Voting Integration 🗳️
**Timeline**: Sprint 5-6 (Weeks 9-12)  
**Goal**: Complete voting transparency with party analysis

### Sprint 5: Voting Data Display

#### ✅ **Task 5.1**: Voting Details Page/Modal
- [ ] **Subtask 5.1.1**: Detailed voting interface
  - Design comprehensive voting breakdown view
  - Show Sejm and Senate votes side-by-side
  - Include individual MP voting records
  - **Estimate**: 3 days
  - **Owner**: Frontend Developer
  - **Dependencies**: None

- [ ] **Subtask 5.1.2**: Party voting visualization
  - Color-coded party breakdown charts
  - Vote count displays with percentages
  - Party discipline indicators
  - **Estimate**: 2 days
  - **Owner**: Frontend Developer
  - **Dependencies**: 5.1.1

- [ ] **Subtask 5.1.3**: Individual voting records
  - Searchable MP/Senator voting table
  - Party affiliation and vote indicators
  - Export functionality for voting data
  - **Estimate**: 2 days
  - **Owner**: Frontend Developer
  - **Dependencies**: 5.1.1

#### ✅ **Task 5.2**: Voting Data API
- [ ] **Subtask 5.2.1**: Voting details endpoint
  - Create `/acts/{id}/votes` endpoint
  - Return comprehensive voting information
  - Include party breakdowns and individual votes
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: Phase 1 completion

- [ ] **Subtask 5.2.2**: Vote analysis functions
  - Calculate party discipline metrics
  - Determine bipartisan support levels
  - Generate voting statistics
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: 5.2.1

### Sprint 6: Real-time Updates & Performance

#### ✅ **Task 6.1**: Real-time Update System
- [ ] **Subtask 6.1.1**: WebSocket implementation
  - Set up WebSocket for real-time updates
  - Push status changes to connected clients
  - Handle connection management
  - **Estimate**: 2 days
  - **Owner**: Full-stack Developer
  - **Dependencies**: None

- [ ] **Subtask 6.1.2**: Update notification system
  - Visual indicators for new updates
  - Toast notifications for important changes
  - Update highlighting on board
  - **Estimate**: 1 day
  - **Owner**: Frontend Developer
  - **Dependencies**: 6.1.1

#### ✅ **Task 6.2**: Performance Optimization
- [ ] **Subtask 6.2.1**: Caching implementation
  - Redis caching for frequently accessed data
  - Cache voting information and statistics
  - Implement cache invalidation strategy
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: None

- [ ] **Subtask 6.2.2**: Database optimization
  - Index optimization for enhanced queries
  - Query performance tuning
  - Database connection pooling
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: 6.2.1

---

## Phase 4: Advanced Features 🚀
**Timeline**: Sprint 7-8 (Weeks 13-16)  
**Goal**: Advanced search, mobile optimization, monitoring

### Sprint 7: Mobile Optimization

#### ✅ **Task 7.1**: Mobile-Responsive Board
- [ ] **Subtask 7.1.1**: Mobile board layout
  - Vertical stack layout for mobile
  - Swipe navigation between sections
  - Touch-optimized interactions
  - **Estimate**: 3 days
  - **Owner**: Frontend Developer
  - **Dependencies**: Phase 2 completion

- [ ] **Subtask 7.1.2**: Compact card design
  - Mobile-optimized card layout
  - Essential information prioritization
  - Expandable details for full info
  - **Estimate**: 2 days
  - **Owner**: Frontend Developer
  - **Dependencies**: 7.1.1

#### ✅ **Task 7.2**: Mobile Voting Interface
- [ ] **Subtask 7.2.1**: Mobile voting details
  - Touch-friendly voting breakdown view
  - Simplified party visualization
  - Swipe through voting stages
  - **Estimate**: 2 days
  - **Owner**: Frontend Developer
  - **Dependencies**: Phase 3 completion

### Sprint 8: Monitoring & Analytics

#### ✅ **Task 8.1**: System Monitoring
- [ ] **Subtask 8.1.1**: Application monitoring
  - Implement comprehensive logging
  - Set up error tracking and alerting
  - Performance monitoring dashboard
  - **Estimate**: 2 days
  - **Owner**: DevOps/Backend Developer
  - **Dependencies**: None

- [ ] **Subtask 8.1.2**: Data pipeline monitoring
  - Monitor API polling success rates
  - Track data quality metrics
  - Alert on missing or stale data
  - **Estimate**: 2 days
  - **Owner**: DevOps/Backend Developer
  - **Dependencies**: 8.1.1

#### ✅ **Task 8.2**: User Analytics
- [ ] **Subtask 8.2.1**: Usage analytics implementation
  - Track user engagement metrics
  - Monitor feature usage patterns
  - A/B testing framework setup
  - **Estimate**: 1 day
  - **Owner**: Full-stack Developer
  - **Dependencies**: None

---

## Phase 5: Data Enrichment 📊
**Timeline**: Sprint 9-10 (Weeks 17-20)  
**Goal**: Fill data gaps and external integrations

### Sprint 9: Gap Filling & External Sources

#### ✅ **Task 9.1**: Presidential Stage Tracking
- [ ] **Subtask 9.1.1**: Presidential website monitoring
  - Implement web scraping for presidential actions
  - Track signing ceremonies and vetoes
  - Parse presidential press releases
  - **Estimate**: 3 days
  - **Owner**: Backend Developer
  - **Dependencies**: None

- [ ] **Subtask 9.1.2**: Timeline-based inference
  - Implement 21-day rule tracking
  - Infer presidential action based on timing
  - Flag Acts requiring presidential action
  - **Estimate**: 1 day
  - **Owner**: Backend Developer
  - **Dependencies**: 9.1.1

#### ✅ **Task 9.2**: Constitutional Court Integration
- [ ] **Subtask 9.2.1**: Court decision monitoring
  - Monitor Constitutional Court website
  - Parse court decisions related to Acts
  - Track constitutional challenges
  - **Estimate**: 2 days
  - **Owner**: Backend Developer
  - **Dependencies**: None

### Sprint 10: Manual Data Entry & Quality Assurance

#### ✅ **Task 10.1**: Administrative Interface
- [ ] **Subtask 10.1.1**: Admin data entry interface
  - Create interface for manual data corrections
  - Allow manual status updates for edge cases
  - Audit trail for manual changes
  - **Estimate**: 2 days
  - **Owner**: Full-stack Developer
  - **Dependencies**: None

#### ✅ **Task 10.2**: Data Quality Assurance
- [ ] **Subtask 10.2.1**: Comprehensive testing
  - End-to-end testing of all features
  - Data accuracy validation
  - Performance testing under load
  - **Estimate**: 3 days
  - **Owner**: QA/Full-stack Developer
  - **Dependencies**: All previous phases

- [ ] **Subtask 10.2.2**: Launch preparation
  - Production deployment preparation
  - User documentation creation
  - Support system setup
  - **Estimate**: 2 days
  - **Owner**: Full-stack Developer
  - **Dependencies**: 10.2.1

---

## Success Metrics & KPIs

### Development Metrics
- [ ] **Code Coverage**: Maintain >80% test coverage
- [ ] **Performance**: Page load times <2 seconds
- [ ] **Reliability**: 99.5% uptime target
- [ ] **Data Accuracy**: <1% user-reported data issues

### Launch Criteria
- [ ] **Feature Completeness**: All Phase 1-4 features implemented
- [ ] **Data Coverage**: 90%+ Acts have enhanced status information
- [ ] **Voting Coverage**: 95%+ Acts have voting breakdowns where available
- [ ] **Mobile Functionality**: All features work on mobile devices
- [ ] **Performance**: Meets all performance requirements

### Post-Launch Success (3 months)
- [ ] **User Engagement**: 50% increase in average session duration
- [ ] **Data Completeness**: 95% of trackable lifecycle stages covered
- [ ] **User Satisfaction**: 4.5+ rating in user feedback
- [ ] **Media Usage**: Evidence of platform citations in media

---

## Risk Management

### High-Risk Items
1. **API Rate Limits**: Sejm API may impose rate limits
   - **Mitigation**: Implement exponential backoff, caching
2. **Data Quality**: Senate-Sejm linking may be imperfect
   - **Mitigation**: Manual override system, user feedback
3. **Performance**: Large voting datasets may impact performance
   - **Mitigation**: Aggressive caching, pagination, lazy loading

### Dependencies
- **External APIs**: Rely on Sejm and Senate data availability
- **Data Format Stability**: API/data format changes could break integration
- **Browser Compatibility**: Advanced features require modern browsers

---

## Resources Required

### Team Composition
- **Backend Developer**: 2 FTE for 20 weeks
- **Frontend Developer**: 1.5 FTE for 20 weeks  
- **Full-stack Developer**: 1 FTE for 20 weeks
- **DevOps/QA**: 0.5 FTE for 20 weeks

### Infrastructure
- **Enhanced Database**: PostgreSQL with additional storage
- **Caching Layer**: Redis for performance optimization
- **Monitoring**: Application and infrastructure monitoring tools
- **CDN**: For improved global performance

---

## Next Steps

1. **Approval**: Get stakeholder approval for PRD and task plan
2. **Team Assembly**: Assign developers to project phases
3. **Environment Setup**: Prepare development and staging environments
4. **Sprint 1 Kickoff**: Begin with database schema enhancements
5. **Weekly Reviews**: Regular progress reviews and adjustments

**Ready to begin implementation when approved! 🚀**