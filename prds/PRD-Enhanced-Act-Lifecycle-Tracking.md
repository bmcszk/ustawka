# PRD: Enhanced Act Lifecycle Tracking System

## Product Overview

Transform Ustawka from a basic Act monitoring tool into a comprehensive Polish legislative transparency platform with detailed Act lifecycle tracking, voting information, and enhanced status management.

## Problem Statement

**Current Limitations:**
- Basic status tracking (only published acts from ELI API)
- Limited board columns and information density
- No parliamentary process visibility
- Missing voting information and party analysis
- Gaps in Senate and Presidential stage tracking
- No real-time legislative process monitoring

**User Needs:**
- Complete Act lifecycle visibility from conception to repeal
- Detailed voting information by party and individual MPs
- Real-time status updates throughout legislative process
- Enhanced filtering and analysis capabilities
- Legislative process transparency and civic engagement

## Product Goals

### Primary Goals
1. **Complete Lifecycle Tracking**: Track Acts through all 9 lifecycle stages
2. **Enhanced Board View**: Rich information display with multiple status columns
3. **Voting Transparency**: Complete Sejm and Senate voting information
4. **Real-time Updates**: Live tracking of legislative progress
5. **Data Enrichment**: Comprehensive Act details from multiple sources

### Success Metrics
- **Coverage**: 90%+ Act lifecycle stages tracked automatically
- **Data Completeness**: 95%+ Acts have complete voting information
- **User Engagement**: 50% increase in session duration
- **Update Frequency**: Real-time updates within 2 hours of legislative events
- **Information Density**: 10+ data points per Act displayed

## User Stories

### Epic 1: Enhanced Act Status Tracking

**As a** policy researcher  
**I want** to see detailed Act status progression through all legislative stages  
**So that** I can understand exactly where each Act is in the process

**As a** journalist  
**I want** to track Acts that are stuck in committee or facing delays  
**So that** I can investigate legislative bottlenecks

**As a** citizen advocate  
**I want** to see which Acts are coming up for votes  
**So that** I can contact my representatives

### Epic 2: Voting Information and Analysis

**As a** political analyst  
**I want** to see how each party voted on specific Acts  
**So that** I can analyze party positions and coalition dynamics

**As a** transparency advocate  
**I want** to see individual MP voting records  
**So that** I can hold representatives accountable

**As a** researcher  
**I want** to compare Sejm and Senate voting patterns  
**So that** I can analyze bicameral legislative dynamics

### Epic 3: Enhanced Board Interface

**As a** legislative monitor  
**I want** a comprehensive board view with multiple status columns  
**So that** I can quickly assess the state of all legislation

**As a** citizen  
**I want** to see rich Act information at a glance  
**So that** I don't need to click through multiple pages

## Detailed Requirements

### Functional Requirements

#### FR1: Enhanced Act Status System

**New Status Categories (replacing simple published/not published):**

**📝 Parliamentary Process Statuses:**
- `submitted` - Act submitted to Sejm
- `committee_first_reading` - First reading in committee
- `committee_work` - Under committee examination
- `second_reading` - Second reading (amendment stage)
- `third_reading` - Final Sejm vote pending
- `passed_sejm` - Passed Sejm, sent to Senate
- `senate_review` - Under Senate examination
- `senate_amended` - Senate proposed amendments
- `senate_rejected` - Senate rejected Act
- `override_vote` - Sejm override vote pending
- `presidential_review` - Awaiting Presidential action
- `presidential_veto` - Presidential veto
- `veto_override` - Veto override attempt

**📋 Publication Statuses:**
- `published` - Published in Dziennik Ustaw
- `in_force` - Act entered into force
- `amended` - Act has been amended
- `repealed` - Act repealed
- `expired` - Act expired/sunset
- `invalidated` - Declared unconstitutional

**⚠️ Special Statuses:**
- `urgent` - Urgent procedure
- `eu_compliance` - EU law implementation
- `constitutional_review` - Constitutional Court review
- `consolidated_available` - Consolidated version available

#### FR2: Enhanced Board Columns

**New Board Layout (Kanban-style with enhanced columns):**

```
┌─────────────────┬──────────────────┬─────────────────┬──────────────────┬─────────────────┐
│    SUBMITTED    │  COMMITTEE WORK  │ SEJM READINGS   │  SENATE REVIEW   │   PUBLISHED     │
├─────────────────┼──────────────────┼─────────────────┼──────────────────┼─────────────────┤
│ • Bills entered │ • First reading  │ • Second reading│ • Senate work    │ • In force      │
│ • Awaiting      │ • Committee      │ • Third reading │ • Amendments     │ • Recently      │
│   assignment    │   examination    │ • Final vote    │ • Decisions      │   published     │
│ • Recent        │ • Committee      │ • Passed Sejm   │ • Override votes │ • Amendments    │
│   submissions   │   reports        │                 │                  │ • Repealed      │
└─────────────────┴──────────────────┴─────────────────┴──────────────────┴─────────────────┘
```

**Additional Status Columns:**
- **PRESIDENTIAL** - Presidential review, veto, signing
- **CONSTITUTIONAL** - Constitutional Court challenges
- **SPECIAL PROCEDURES** - Urgent bills, EU implementation

#### FR3: Enhanced Act Information Display

**Act Card Information (displayed on each card):**

```
┌─────────────────────────────────────────────────────────┐
│ 📋 Act Title (truncated with tooltip)                   │
├─────────────────────────────────────────────────────────┤
│ 🏛️ DU/2024/1234 • Type: Ustawa • Year: 2024            │
│ 📅 Submitted: 2024-03-15 • Updated: 2024-06-20         │
│ ⏰ Current Stage: Committee Work (15 days)              │
├─────────────────────────────────────────────────────────┤
│ 🗳️ VOTING INFO:                                        │
│ • Sejm: PiS 177-YES, KO 152-NO (click for details)     │
│ • Senate: Pending review                                │
├─────────────────────────────────────────────────────────┤
│ 👥 STAKEHOLDERS:                                        │
│ • Initiator: Government/Deputies/Citizens               │
│ • Committee: GOR (Gospodarki i Rozwoju)                │
│ • Rapporteur: Jan Kowalski (PiS)                       │
├─────────────────────────────────────────────────────────┤
│ 🏷️ Tags: [urgent] [eu-law] [budget-impact]              │
│ 🔗 Links: [PDF] [Sejm Process] [Voting Details]        │
└─────────────────────────────────────────────────────────┘
```

#### FR4: Data Integration Strategy

**Primary Data Sources:**

1. **Sejm API** (`api.sejm.gov.pl`)
   - Process tracking: `/sejm/term{X}/processes/{number}`
   - Voting data: `/sejm/term{X}/votings/{proceeding}/{voting}`
   - Prints/bills: `/sejm/term{X}/prints`

2. **ELI API** (`api.sejm.gov.pl/eli`)
   - Published acts: `/eli/acts/DU/{year}`
   - Act details: `/eli/acts/{publisher}/{year}/{position}`
   - Status changes: `/eli/changes/acts`

3. **Senate Open Data** (`dane.gov.pl`)
   - Voting records: Dataset 4648
   - Individual and party voting breakdowns

4. **Enrichment Sources:**
   - RCL Legislative Portal links
   - EU law compliance indicators
   - Amendment tracking through references

**Data Refresh Strategy:**
- **Real-time**: Sejm API polling every 30 minutes
- **Daily**: Senate data updates
- **Weekly**: ELI API full synchronization
- **Event-driven**: Process status changes trigger immediate updates

#### FR5: Voting Information Integration

**Voting Data Display:**

**Detailed Voting Modal/Page:**
```
┌─────────────────────────────────────────────────────────┐
│ 🗳️ VOTING DETAILS: Tax Ordinance Amendment              │
├─────────────────────────────────────────────────────────┤
│ SEJM VOTE (2024-06-04, Third Reading)                   │
│ Result: PASSED (255 NO, 181 YES, 1 ABSTAIN)            │
│                                                         │
│ Party Breakdown:                                        │
│ 🔴 PiS (Opposition): 177 YES, 12 ABSENT                │
│ 🔵 KO (Government): 152 NO, 5 ABSENT                   │
│ 🟢 PSL-TD: 30 NO, 2 ABSENT                             │
│ 🟡 Lewica: 19 NO, 2 ABSENT                             │
│ 🟣 Polska2050-TD: 30 NO, 2 ABSENT                      │
│ ⚫ Konfederacja: 15 NO, 1 YES                           │
│                                                         │
│ [View Individual MP Votes] [Download CSV]              │
├─────────────────────────────────────────────────────────┤
│ SENATE VOTE (Pending)                                   │
│ Expected: 2024-07-05                                    │
└─────────────────────────────────────────────────────────┘
```

#### FR6: Advanced Filtering and Search

**Enhanced Filter Options:**
- **Status**: Multiple status selection
- **Year**: Range selection
- **Initiator**: Government/Deputy/Citizen/Senate/President
- **Committee**: All committees with counts
- **Voting Stage**: Has voted/Pending vote/Multiple votes
- **Party Support**: Government supported/Opposition supported/Bipartisan
- **Timeline**: Submitted date range, Days in current stage
- **Type**: Ustawa/Rozporządzenie/Uchwała
- **Tags**: Urgent/EU law/Budget impact/Constitutional

**Search Functionality:**
- Full-text search in titles and descriptions
- ELI ID search
- Print number search
- MP/Senator name search for voting records

### Non-Functional Requirements

#### Performance Requirements
- **Page Load**: Initial board load < 2 seconds
- **Data Updates**: Process status updates within 30 minutes
- **Voting Data**: Party breakdowns load < 1 second
- **Search**: Search results return < 500ms
- **Concurrent Users**: Support 1000+ concurrent users

#### Scalability Requirements
- **Data Volume**: Handle 10,000+ Acts per year
- **API Calls**: Rate-limited API integration (max 100 req/min)
- **Storage**: Efficient storage for voting records (100K+ votes/year)
- **Caching**: Redis caching for frequently accessed data

#### Reliability Requirements
- **Uptime**: 99.5% uptime
- **Data Accuracy**: 99%+ accuracy in status tracking
- **Error Handling**: Graceful degradation when APIs are unavailable
- **Monitoring**: Real-time monitoring of data pipelines

### Technical Requirements

#### Backend Enhancement

**Database Schema Additions:**

```sql
-- Enhanced Act table
ALTER TABLE acts ADD COLUMN detailed_status VARCHAR(50);
ALTER TABLE acts ADD COLUMN current_stage VARCHAR(100);
ALTER TABLE acts ADD COLUMN stage_date TIMESTAMP;
ALTER TABLE acts ADD COLUMN days_in_stage INTEGER;
ALTER TABLE acts ADD COLUMN initiator_type VARCHAR(50);
ALTER TABLE acts ADD COLUMN committee_code VARCHAR(10);
ALTER TABLE acts ADD COLUMN rapporteur_name VARCHAR(100);
ALTER TABLE acts ADD COLUMN urgency_status VARCHAR(20);
ALTER TABLE acts ADD COLUMN eu_compliance BOOLEAN;
ALTER TABLE acts ADD COLUMN process_print_number VARCHAR(20);
ALTER TABLE acts ADD COLUMN rcl_link VARCHAR(255);

-- New voting table
CREATE TABLE act_votes (
    id SERIAL PRIMARY KEY,
    act_id VARCHAR(50) REFERENCES acts(id),
    chamber VARCHAR(10), -- 'sejm' or 'senate'
    vote_date TIMESTAMP,
    vote_type VARCHAR(50), -- 'first_reading', 'amendment', 'final_passage'
    total_voted INTEGER,
    yes_votes INTEGER,
    no_votes INTEGER,
    abstain_votes INTEGER,
    absent_votes INTEGER,
    result VARCHAR(20), -- 'passed', 'failed'
    voting_data JSONB, -- Full voting details
    created_at TIMESTAMP DEFAULT NOW()
);

-- Party voting breakdowns
CREATE TABLE party_votes (
    id SERIAL PRIMARY KEY,
    vote_id INTEGER REFERENCES act_votes(id),
    party_name VARCHAR(100),
    total_members INTEGER,
    yes_votes INTEGER,
    no_votes INTEGER,
    abstain_votes INTEGER,
    absent_votes INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Process stages tracking
CREATE TABLE act_stages (
    id SERIAL PRIMARY KEY,
    act_id VARCHAR(50) REFERENCES acts(id),
    stage_name VARCHAR(100),
    stage_date TIMESTAMP,
    committee_code VARCHAR(10),
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**API Enhancements:**

```go
// Enhanced Act structure
type EnhancedAct struct {
    // Existing fields
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Status      string    `json:"status"`
    Published   string    `json:"published"`
    
    // New fields
    DetailedStatus     string               `json:"detailed_status"`
    CurrentStage       string               `json:"current_stage"`
    StageDate          time.Time            `json:"stage_date"`
    DaysInStage        int                  `json:"days_in_stage"`
    InitiatorType      string               `json:"initiator_type"`
    CommitteeCode      string               `json:"committee_code"`
    RapporteurName     string               `json:"rapporteur_name"`
    UrgencyStatus      string               `json:"urgency_status"`
    EUCompliance       bool                 `json:"eu_compliance"`
    ProcessPrintNumber string               `json:"process_print_number"`
    RCLLink           string               `json:"rcl_link"`
    
    // Voting information
    SejmVotes          []VotingRecord       `json:"sejm_votes"`
    SenateVotes        []VotingRecord       `json:"senate_votes"`
    PartyBreakdowns    map[string]PartyVote `json:"party_breakdowns"`
    
    // Lifecycle tracking
    Stages             []ProcessStage       `json:"stages"`
    
    // Metadata
    Tags               []string             `json:"tags"`
    Links              ActLinks             `json:"links"`
}

type VotingRecord struct {
    Date        time.Time    `json:"date"`
    VoteType    string       `json:"vote_type"`
    Result      string       `json:"result"`
    TotalVoted  int          `json:"total_voted"`
    YesVotes    int          `json:"yes_votes"`
    NoVotes     int          `json:"no_votes"`
    AbstainVotes int         `json:"abstain_votes"`
    PartyBreakdown map[string]PartyVote `json:"party_breakdown"`
}

type PartyVote struct {
    Party        string `json:"party"`
    TotalMembers int    `json:"total_members"`
    YesVotes     int    `json:"yes_votes"`
    NoVotes      int    `json:"no_votes"`
    AbstainVotes int    `json:"abstain_votes"`
    AbsentVotes  int    `json:"absent_votes"`
}

type ProcessStage struct {
    StageName    string    `json:"stage_name"`
    StageDate    time.Time `json:"stage_date"`
    CommitteeCode string   `json:"committee_code"`
    Notes        string    `json:"notes"`
}

type ActLinks struct {
    PDFDocument  string `json:"pdf_document"`
    SejmProcess  string `json:"sejm_process"`
    VotingDetails string `json:"voting_details"`
    RCLPortal    string `json:"rcl_portal"`
}
```

#### Frontend Enhancement

**Enhanced Board Component:**

```go
// Enhanced board view with multiple columns
type EnhancedBoardView struct {
    Columns []StatusColumn `json:"columns"`
    Filters FilterOptions  `json:"filters"`
    Search  SearchOptions  `json:"search"`
}

type StatusColumn struct {
    ID          string         `json:"id"`
    Title       string         `json:"title"`
    Statuses    []string       `json:"statuses"`
    Acts        []EnhancedAct  `json:"acts"`
    Count       int            `json:"count"`
    Color       string         `json:"color"`
}

type FilterOptions struct {
    Years        []int          `json:"years"`
    Statuses     []string       `json:"statuses"`
    Initiators   []string       `json:"initiators"`
    Committees   []string       `json:"committees"`
    VotingStage  []string       `json:"voting_stage"`
    PartySupport []string       `json:"party_support"`
    Timeline     TimelineFilter `json:"timeline"`
    Types        []string       `json:"types"`
    Tags         []string       `json:"tags"`
}
```

### UI/UX Requirements

#### Enhanced Board Interface

**Layout Design:**
- **Responsive Kanban Board**: 5-7 status columns with horizontal scrolling
- **Card Information Density**: 10+ data points per card
- **Quick Actions**: Voting details, PDF view, external links
- **Color Coding**: Status-based colors, urgency indicators
- **Progressive Disclosure**: Summary view with expandable details

**Voting Information Display:**
- **Party Vote Indicators**: Color-coded party positions
- **Vote Result Badges**: Passed/Failed with vote counts
- **Quick Stats**: Voting participation rates, party discipline
- **Drill-down Capability**: Individual MP voting records

#### Mobile Optimization

**Mobile Board View:**
- **Vertical Stack Layout**: Status sections stack vertically
- **Swipe Navigation**: Swipe between status sections
- **Compact Cards**: Essential information only
- **Touch Optimized**: Larger touch targets, gesture support

### Data Enrichment Strategy

#### Gap Filling Approach

**Senate Stage Data:**
- **Primary**: Senate Open Data Portal (comprehensive voting)
- **Secondary**: Web scraping Senate website for non-voting updates
- **Tertiary**: Manual data entry for critical gaps

**Presidential Stage Data:**
- **Primary**: Monitor presidential website for signing ceremonies
- **Secondary**: Government announcements and press releases
- **Tertiary**: Timeline-based inference (21-day rule)

**Constitutional Court Data:**
- **Primary**: Constitutional Court website monitoring
- **Secondary**: Legal database integration
- **Tertiary**: Media monitoring for court decisions

**Amendment Tracking:**
- **Primary**: ELI API references and consolidated texts
- **Secondary**: Cross-reference with new legislation
- **Tertiary**: Manual legal analysis

### Implementation Plan

#### Phase 1: Data Infrastructure (Sprint 1-2)
- Enhanced database schema
- Sejm API integration enhancement
- Senate data integration
- Basic status enrichment

#### Phase 2: Enhanced Board (Sprint 3-4)
- New status columns implementation
- Enhanced Act card design
- Basic voting information display
- Improved filtering system

#### Phase 3: Voting Integration (Sprint 5-6)
- Complete voting data integration
- Party breakdown visualization
- Individual voting records
- Voting analysis features

#### Phase 4: Advanced Features (Sprint 7-8)
- Advanced search functionality
- Real-time updates
- Mobile optimization
- Performance optimization

#### Phase 5: Data Enrichment (Sprint 9-10)
- Senate/Presidential gap filling
- External data source integration
- Manual data entry tools
- Data quality monitoring

### Success Criteria

#### Launch Criteria
- ✅ All current Acts have enhanced status information
- ✅ 90%+ Acts have complete voting breakdowns where available
- ✅ New board interface supports 5+ status columns
- ✅ Mobile interface is fully functional
- ✅ Page load times meet performance requirements
- ✅ Data pipeline successfully updates every 30 minutes

#### Post-Launch Success Metrics (3 months)
- **User Engagement**: 50% increase in average session duration
- **Data Coverage**: 95% of trackable lifecycle stages covered
- **User Satisfaction**: 4.5+ rating in user feedback
- **Performance**: 99%+ uptime, <2s page loads
- **Data Accuracy**: <1% user-reported data issues

#### Long-term Goals (6-12 months)
- **Complete Transparency**: Track 100% of Act lifecycle where data exists
- **Civic Engagement**: 10K+ monthly active users
- **Media Integration**: Regular media citations of platform data
- **API Usage**: External developers using API for civic apps
- **Policy Impact**: Evidence of platform influence on legislative transparency

This PRD provides a comprehensive roadmap for transforming Ustawka into a world-class legislative transparency platform with unprecedented detail and real-time tracking capabilities.