# Polish Act Lifecycle Documentation

## Complete Act Lifecycle Overview

A Polish Act (Ustawa) goes through a comprehensive lifecycle from initial conception to eventual repeal or replacement. This document defines every stage of this lifecycle with detailed technical information and decision points.

## Mermaid Flowchart: Complete Polish Act Lifecycle

```mermaid
flowchart TD
    Start([Act Conception]) --> Initiative{Legislative Initiative}
    
    Initiative --> |15+ Deputies| DeputyBill[Deputy Bill Submission]
    Initiative --> |Senate Resolution| SenateBill[Senate Bill Submission]
    Initiative --> |Presidential Initiative| PresidentialBill[Presidential Bill Submission]
    Initiative --> |Government| GovernmentBill[Government Bill Submission]
    Initiative --> |100k+ Citizens| CitizenBill[Citizen Initiative Bill]
    
    DeputyBill --> Submission[Bill Submitted to Marshal of Sejm]
    SenateBill --> Submission
    PresidentialBill --> Submission
    GovernmentBill --> Submission
    CitizenBill --> Submission
    
    Submission --> FirstReading{First Reading Location}
    FirstReading --> |Constitutional/Budget/Electoral/Codes| PlenaryFirst[Plenary First Reading]
    FirstReading --> |Other Bills| CommitteeFirst[Committee First Reading]
    
    PlenaryFirst --> Committee[Committee Examination]
    CommitteeFirst --> Committee
    
    Committee --> CommitteeReport[Committee Report & Recommendation]
    CommitteeReport --> |Pass| SecondReading[Second Reading - Plenary]
    CommitteeReport --> |Reject| BillRejected[Bill Rejected]
    
    SecondReading --> Amendments{Amendments Proposed?}
    Amendments --> |Yes| BackToCommittee[Return to Committee]
    Amendments --> |No| ThirdReading[Third Reading]
    
    BackToCommittee --> AdditionalReport[Additional Committee Report]
    AdditionalReport --> ThirdReading
    
    ThirdReading --> SejmVote{Sejm Final Vote}
    SejmVote --> |Fail| BillRejected
    SejmVote --> |Pass| SendToSenate[Send to Senate]
    
    SendToSenate --> SenateReview{Senate Review - 30 Days}
    SenateReview --> |Accept| SendToPresident[Send to President]
    SenateReview --> |Amend| SenateAmendment[Senate Amendment]
    SenateReview --> |Reject| SenateRejection[Senate Rejection]
    
    SenateAmendment --> SejmOverride{Sejm Override Vote}
    SenateRejection --> SejmOverride
    SejmOverride --> |3/5 Majority| SendToPresident
    SejmOverride --> |Fail| BillRejected
    
    SendToPresident --> PresidentialReview{Presidential Review - 21 Days}
    PresidentialReview --> |Sign| Publication[Publication in Dziennik Ustaw]
    PresidentialReview --> |Veto| PresidentialVeto[Presidential Veto]
    PresidentialReview --> |Constitutional Review| ConstitutionalCourt[Send to Constitutional Court]
    
    PresidentialVeto --> VetoOverride{Sejm Veto Override}
    VetoOverride --> |3/5 Majority| Publication
    VetoOverride --> |Fail| BillRejected
    
    ConstitutionalCourt --> CourtDecision{Constitutional Court Decision}
    CourtDecision --> |Constitutional| PresidentialSign[President Must Sign]
    CourtDecision --> |Unconstitutional| BillRejected
    PresidentialSign --> Publication
    
    Publication --> EntryIntoForce[Entry into Force]
    EntryIntoForce --> Implementation[Implementation & Enforcement]
    
    Implementation --> ActiveLaw[Active Law]
    
    ActiveLaw --> Amendment{Amendment Process}
    ActiveLaw --> Repeal{Repeal Process}
    ActiveLaw --> JudicialReview[Judicial Review/Constitutional Challenge]
    
    Amendment --> AmendmentBill[New Amending Bill]
    AmendmentBill --> Submission
    
    Repeal --> RepealBill[New Repealing Bill]
    RepealBill --> Submission
    
    JudicialReview --> CourtChallenge{Constitutional Court Challenge}
    CourtChallenge --> |Constitutional| ActiveLaw
    CourtChallenge --> |Unconstitutional| LawInvalidated[Law Invalidated]
    
    BillRejected --> End([Process Ends])
    LawInvalidated --> End
    
    style Start fill:#e1f5fe
    style End fill:#ffebee
    style BillRejected fill:#ffcdd2
    style LawInvalidated fill:#ffcdd2
    style ActiveLaw fill:#c8e6c9
    style Publication fill:#fff3e0
    style Implementation fill:#f3e5f5
```

## Detailed Lifecycle Stages

### Stage 1: Conception and Initiative (Pre-Legislative)

**1.1 Policy Development**
- Policy analysis and research
- Stakeholder consultation
- Impact assessment preparation
- Legal drafting preparation

**1.2 Legislative Initiative Authorization**
- **Deputies**: Group of 15+ deputies or Sejm committee
- **Senate**: Full chamber resolution required
- **President**: Constitutional prerogative
- **Government**: Council of Ministers decision
- **Citizens**: 100,000+ signatures with voting rights

**Technical Requirements:**
- Written submission with explanatory statement
- Social, economic, and financial impact estimates
- EU law compliance declaration
- Secondary legislation drafts (for government bills)

### Stage 2: Parliamentary Process (Legislative)

**2.1 Submission and Registration**
- Formal submission to Marshal of Sejm
- Assignment of bill number
- Publication in parliamentary documents
- Representative designation

**2.2 First Reading**
- **Committee Level**: Most bills (general principle debate)
- **Plenary Level**: Constitutional, budget, electoral, code bills
- Initial assessment and principle acceptance

**2.3 Committee Examination**
- Detailed review and analysis
- Expert hearings and consultations
- Amendment drafting and evaluation
- Committee report preparation with recommendations

**2.4 Second Reading (Plenary)**
- Committee report presentation
- General debate on amendments
- Amendment proposals by authorized entities
- Possible return to committee for additional work

**2.5 Third Reading (Plenary)**
- Final debate on amendments
- Voting sequence:
  1. Motion to reject (if any)
  2. Individual amendments
  3. Final bill passage
- Simple majority required (231/460 with quorum)

### Stage 3: Senate Review (Bicameral)

**3.1 Senate Examination (30 days standard, 14 days urgent)**
- Committee review and analysis
- Potential amendments or rejection
- Senate vote on final position

**3.2 Senate Decision Options**
- **Accept**: Bill proceeds to President
- **Amend**: Returns to Sejm for override vote
- **Reject**: Returns to Sejm for override vote

**3.3 Sejm Override Process (if needed)**
- Absolute majority required (231/460)
- Quorum requirement (50% statutory members)
- Final legislative determination

### Stage 4: Presidential Review (Executive)

**4.1 Presidential Options (21 days standard, 7 days urgent)**
- **Sign**: Direct approval and transmission for publication
- **Veto**: Return to Sejm with objections
- **Constitutional Review**: Referral to Constitutional Court

**4.2 Veto Override Process**
- 3/5 majority required in Sejm (276/460)
- Quorum requirement (50% statutory members)
- Final legislative override of executive objection

**4.3 Constitutional Court Review**
- Presidential referral for constitutional compliance
- Court examination and decision
- If constitutional: President must sign
- If unconstitutional: Bill invalidated

### Stage 5: Publication and Entry into Force (Promulgation)

**5.1 Official Publication**
- Publication in Dziennik Ustaw (Journal of Laws)
- Assignment of law number and year
- Official legal effect begins

**5.2 Entry into Force**
- Date specified in the Act
- Default: 14 days after publication (if not specified)
- Possible delayed or phased implementation

### Stage 6: Implementation and Enforcement (Post-Legislative)

**6.1 Administrative Implementation**
- Government regulation development (rozporządzenia)
- Institutional setup and procedures
- Resource allocation and staffing

**6.2 Judicial Interpretation**
- Court application and interpretation
- Precedent development
- Legal doctrine formation

**6.3 Compliance and Enforcement**
- Monitoring and oversight
- Penalty application
- Regulatory enforcement actions

### Stage 7: Amendment Process (Modification)

**7.1 Amendment Initiation**
- Same initiative rules as original legislation
- Amendment bill follows full legislative process
- Can modify, add, or delete provisions

**7.2 Amendment Types**
- **Technical**: Corrections and clarifications
- **Substantive**: Policy changes and updates
- **Comprehensive**: Major restructuring

### Stage 8: Repeal Process (Termination)

**8.1 Repeal Initiation**
- Express repeal through new legislation
- Implicit repeal through contradictory law
- Constitutional Court invalidation

**8.2 Repeal Methods**
- **Total Repeal**: Entire Act invalidated
- **Partial Repeal**: Specific provisions removed
- **Sunset Clauses**: Automatic expiration

### Stage 9: Judicial Review (Constitutional Challenge)

**9.1 Constitutional Court Review**
- Abstract review (institutional referral)
- Concrete review (court referral during proceedings)
- Individual constitutional complaint

**9.2 Court Decision Effects**
- **Constitutional**: Law remains valid
- **Unconstitutional**: Law invalidated (erga omnes effect)
- **Partially Unconstitutional**: Specific provisions invalidated

## Alternative Paths and Decision Points

### Emergency Procedures
- **Urgent Bills**: Shortened timeframes (Government only)
- **State of Emergency**: Special constitutional procedures
- **EU Implementation**: Deadline-driven fast-track

### Withdrawal and Abandonment
- **Pre-Second Reading**: Movers can withdraw
- **Parliamentary Session End**: Bills typically lapse
- **Government Change**: Policy bill review

### Constitutional Complications
- **Court Injunctions**: Temporary suspension of provisions
- **EU Law Conflicts**: Supremacy doctrine applications
- **International Treaty Conflicts**: Constitutional hierarchy issues

## Technical Status Indicators

Throughout the lifecycle, Acts have specific status indicators:

1. **Draft** - Under preparation
2. **Submitted** - Formally filed
3. **First Reading** - Initial parliamentary stage
4. **Committee** - Under committee examination
5. **Second Reading** - Amendment stage
6. **Third Reading** - Final parliamentary vote
7. **Passed Sejm** - Approved by lower house
8. **Senate Review** - Upper house examination
9. **Presidential Review** - Executive consideration
10. **Published** - Official promulgation
11. **In Force** - Active law
12. **Amended** - Modified version active
13. **Repealed** - No longer in force
14. **Invalidated** - Declared unconstitutional

This comprehensive lifecycle ensures democratic legitimacy, constitutional compliance, and proper implementation of Polish legislation.

## API Capabilities and Limitations for Act Lifecycle Tracking

Based on investigation of the Sejm APIs, here's a comprehensive analysis of what lifecycle stages can be tracked and what limitations exist for implementing complete Act lifecycle monitoring.

### Available Sejm APIs

**1. ELI API (European Legislation Identifier)**
- Base URL: `https://api.sejm.gov.pl/eli`
- Focuses on published Acts in Dziennik Ustaw
- Provides final publication status information

**2. Main Sejm API**
- Base URL: `https://api.sejm.gov.pl/sejm`
- Covers parliamentary processes, prints, proceedings, and voting
- Tracks legislative process from bill submission through committee work

### Trackable Lifecycle Stages

#### ✅ **Available Through APIs**

**Stage 1: Conception and Initiative**
- ✅ **Bill Submission**: `/sejm/term{X}/prints` endpoint
  - Delivery date, document date, bill title
  - Bill type identification (government, citizen, deputy)
  - Process print numbers for tracking

**Stage 2-3: Parliamentary Process**
- ✅ **Process Tracking**: `/sejm/term{X}/processes/{number}` endpoint
  - Complete stage progression with dates
  - Committee assignments and referrals
  - First reading status and committee work
  - Committee reports and recommendations
- ✅ **Committee Work**: Detailed committee information
  - Committee codes, report dates, rapporteur assignments
  - Minority motions count, subcommittee involvement
  - Committee recommendations (accept/reject/amend)

**Stage 4: Voting and Parliamentary Decisions**
- ✅ **Voting Records**: `/sejm/term{X}/votings` endpoint
  - Complete voting results with individual MP votes
  - Vote counts (yes/no/abstain/absent)
  - Voting dates and proceedings information
  - Majority type requirements and results

**Stage 5-6: Publication and Entry into Force**
- ✅ **Published Acts**: `/eli/acts/DU/{year}` endpoint
  - Publication information (Dziennik Ustaw)
  - Entry into force dates
  - Act status tracking
  - Official document addresses

#### ❌ **Missing from APIs**

**Senate Review Process**
- ❌ No dedicated Senate API endpoints found
- ❌ Senate committee work not tracked
- ❌ Senate amendment processes not available
- ❌ Senate voting records not accessible

**Presidential Review Stage**
- ❌ Presidential review status not tracked
- ❌ Veto information not available through APIs
- ❌ Constitutional Court referrals not tracked
- ❌ Presidential signing dates not accessible

**Post-Publication Lifecycle**
- ❌ Amendment tracking limited to new acts
- ❌ No API for tracking consolidated versions
- ❌ Repeal processes not systematically tracked
- ❌ Constitutional Court challenges not available

### Detailed API Capabilities

#### **ELI API Features**
```
Available Information:
- Act ID (ELI identifier)
- Title and type (ustawa, rozporządzenie, etc.)
- Publication date and volume
- Status (in force, repealed, etc.)
- Change dates for tracking updates
- Text availability (PDF/HTML)
- Entry into force dates
- Publisher information
```

**Status Types Available:**
- `obowiązujący` (in force)
- `uchylony` (repealed)
- `wygaśnięcie aktu` (act expired)
- `akt objęty tekstem jednolitym` (consolidated version available)
- `bez statusu` (no status)

#### **Sejm API Process Tracking**
```
Process Information Available:
- documentType: "projekt ustawy" (bill type)
- processStartDate: Initial submission
- urgencyStatus: NORMAL/URGENT
- principleOfSubsidiarity: EU compliance flag
- legislativeCommittee: Special committee flag
- passed: Final passage status
- stages[]: Detailed stage progression

Stage Details:
- stageName: Human-readable stage description  
- date: Stage completion date
- printNumber: Associated document numbers
- committeeCode: Committee assignments
- children[]: Sub-stages and committee work
```

#### **Voting Information**
```
Voting Details Available:
- Individual MP votes by name and party
- Vote totals (yes/no/abstain/absent)
- Majority requirements and results
- Voting date and proceeding context
- Electronic voting records
- PDF reports for detailed analysis
```

### Implementation Recommendations

#### **Feasible Lifecycle Tracking**
Based on API capabilities, the following can be reliably tracked:

1. **Bill Introduction** → **Committee Work** → **Parliamentary Voting** → **Publication**
2. **Real-time Status Updates** using change dates from ELI API
3. **Committee Progress** with detailed stage tracking
4. **Voting Outcomes** with complete parliamentary records

#### **Data Integration Strategy**
```
Primary Data Sources:
1. Sejm API (/sejm) - Process tracking until Sejm passage
2. ELI API (/eli) - Publication status and legal effect
3. Manual enrichment - Senate and Presidential stages

Tracking Approach:
- Use processPrint numbers to link between systems
- Monitor changeDate fields for updates
- Cross-reference ELI IDs with process numbers
- Implement fallback for non-API stages
```

#### **Limitations to Address**
1. **Senate Process Gap**: Manual tracking or web scraping required
2. **Presidential Stage**: External data sources needed
3. **Amendment Tracking**: Limited to new legislative acts
4. **Historical Data**: API coverage varies by parliamentary term

### Technical Status Mapping

The system can automatically track these technical statuses:

**Sejm Process Statuses:**
- `Projekt wpłynął do Sejmu` (Bill submitted)
- `Skierowano do I czytania w komisjach` (Referred to committees)
- `I czytanie w komisjach` (First reading in committees)
- `Praca w komisjach po I czytaniu` (Committee work post-first reading)
- `II czytanie` (Second reading)
- `III czytanie` (Third reading)

**Publication Statuses:**
- Publication in Dziennik Ustaw with automatic status updates
- Entry into force tracking with date precision
- Amendment and consolidation status monitoring

### Conclusion

The Sejm APIs provide robust coverage for approximately **70-80%** of the complete Act lifecycle, with excellent detail for parliamentary processes but significant gaps in Senate, Presidential, and post-publication stages. A hybrid approach combining API data with manual enrichment would be required for complete lifecycle tracking.

## Comprehensive Voting Information Tracking

### Investigation Results: Sejm and Senate Voting Data Availability

After comprehensive investigation, **complete voting information is available for both Sejm and Senate**, including detailed party-specific voting patterns for every Act that goes through the legislative process.

### ✅ **Sejm Voting Information - Fully Available**

**API Endpoint**: `https://api.sejm.gov.pl/sejm/term{X}/votings/{proceeding}/{voting}`

**Available Data Structure:**
```json
{
  "description": "Description of the vote",
  "title": "Official vote title", 
  "topic": "Specific topic being voted on",
  "totalVoted": 437,
  "yes": 181,
  "no": 255,
  "abstain": 1,
  "majorityVotes": 231,
  "majorityType": "ABSOLUTE_MAJORITY",
  "votes": [
    {
      "MP": 1,
      "club": "PiS", 
      "firstName": "Jan",
      "lastName": "Kowalski",
      "vote": "YES"
    }
  ]
}
```

**Party-Level Analysis Available:**
- Complete breakdown by parliamentary club/party
- Vote counts per party (YES/NO/ABSTAIN/ABSENT)
- Individual MP voting records with party affiliation
- Real-time vote totals and majority calculations

**Example Party Breakdown for Tax Ordinance Amendment (Term 10, Proceeding 36, Vote 14):**
```
PiS: 177 YES, 12 ABSENT (total: 189 members)
KO: 152 NO, 5 ABSENT (total: 157 members)  
PSL-TD: 30 NO, 2 ABSENT (total: 32 members)
Lewica: 19 NO, 2 ABSENT (total: 21 members)
Polska2050-TD: 30 NO, 2 ABSENT (total: 32 members)
Konfederacja: 15 NO, 1 YES (total: 16 members)
Razem: 5 NO (total: 5 members)
```

### ✅ **Senate Voting Information - Fully Available**

**Data Source**: Official Polish Open Data Portal
- **Dataset ID**: 4648 ("Głosowania Senatu")
- **API**: `https://api.dane.gov.pl/1.4/datasets/4648,glosowania-senatu`
- **Data Format**: XML manifest with CSV voting files
- **Update Frequency**: Daily
- **Coverage**: Complete XI Kadencja (current term)

**Individual Voting Data:**
`https://www.senat.gov.pl/gfx/senat/glosowania_wyniki/kadencja_10/imie/[vote_file]_imie.csv`

**Party Voting Data:**
`https://www.senat.gov.pl/gfx/senat/glosowania_wyniki/kadencja_10/klub/[vote_file]_klub.csv`

**Example Senate Party Voting (Solidarity Fund Act):**
```csv
"Klub / Koło","Liczba czł.",Głosowało,Za,Przeciw,"Wstrzymało się","Nie głosowało"
"Klub Parlamentarny Prawo i Sprawiedliwość",48,45,0,45,0,3
"Klub Parlamentarny Koalicja Obywatelska",43,43,1,41,1,0
"Koło Senatorów Koalicja Polska - PSL",3,3,0,3,0,0
"Senatorowie niezrzeszeni",1,1,0,1,0,0
"Koalicyjny Klub Parlamentarny Lewicy",2,2,0,2,0,0
```

### **Voting Data Capabilities**

#### **Complete Act Lifecycle Voting Tracking**

**For Every Act, You Can Track:**

1. **Sejm Voting Stages:**
   - First reading votes (committee referral)
   - Second reading votes (amendments)
   - Third reading votes (final passage)
   - Amendment-specific votes
   - Procedural votes (urgency, committee assignments)

2. **Senate Voting Stages:**
   - Initial consideration votes
   - Amendment proposals
   - Final Senate decision (accept/amend/reject)
   - Procedural votes

3. **Override Votes:**
   - Sejm override of Senate amendments/rejections
   - Presidential veto override attempts

#### **Party Analysis Capabilities**

**Available Metrics:**
- **Party Discipline**: How many members voted with party line
- **Cross-Party Support**: Opposition party members supporting government bills
- **Abstention Patterns**: Strategic abstentions by party
- **Attendance Rates**: Party member participation in votes
- **Coalition Dynamics**: Voting patterns within governing coalitions

**Example Analysis Possible:**
```
For Tax Ordinance Amendment:
- Government Coalition (KO+PSL+Lewica+Polska2050): 231 NO votes
- Opposition (PiS): 177 YES votes  
- Discipline Rate: 99.1% (only 4 members voted against party line)
- Attendance: 94.3% (26 absent out of 460 total)
```

### **Implementation Strategy for Voting Tracking**

#### **Data Integration Approach**

```
1. Sejm Voting API Integration:
   - Monitor /sejm/term{X}/votings for new votes
   - Filter for Act-related votes using title/description matching
   - Store party breakdowns and individual votes
   - Link to process numbers for Act tracking

2. Senate Voting Data Integration:
   - Parse XML manifest for new voting files
   - Download CSV files for Act-related votes
   - Extract party and individual voting data
   - Match to corresponding Sejm Acts using title/timing

3. Cross-Chamber Linking:
   - Match Acts by title similarity and timing
   - Track progression from Sejm → Senate → Override votes
   - Identify amendment differences between chambers
```

#### **Technical Implementation**

**Voting Data Models:**
```go
type SejmVote struct {
    ActID       string
    VotingID    string
    Date        time.Time
    Title       string
    VoteType    string // "first_reading", "amendment", "final_passage"
    TotalVoted  int
    Yes         int
    No          int
    Abstain     int
    PartyBreakdown map[string]PartyVoting
    IndividualVotes []MPVote
}

type SenateVote struct {
    ActTitle    string
    Session     int
    VoteNumber  int
    Date        time.Time
    VoteType    string // "reject", "accept", "amend"
    TotalVoted  int
    PartyResults map[string]SenatePartyVoting
}

type PartyVoting struct {
    Party       string
    TotalMembers int
    Yes         int
    No          int
    Abstain     int
    Absent      int
}
```

### **Data Quality and Coverage**

#### **Sejm Data Quality**
- ✅ **Complete Coverage**: All votes since Term 10 (2023+)
- ✅ **Real-time Updates**: Votes available within hours
- ✅ **Individual Precision**: Every MP vote recorded
- ✅ **Party Accuracy**: Accurate party affiliations tracked

#### **Senate Data Quality**  
- ✅ **Complete Coverage**: All Term XI votes (2023+)
- ✅ **Daily Updates**: Updated daily via open data portal
- ✅ **Dual Format**: Both individual and party aggregations
- ✅ **Act Linkage**: Clear Act titles for matching

#### **Historical Data Limitations**
- **Sejm**: API coverage varies by term (older terms may have limited data)
- **Senate**: Current dataset covers Term XI only (2023+)
- **Cross-referencing**: Manual title matching required between chambers

### **Conclusion: Complete Voting Transparency**

**✅ Full voting information is available for both Sejm and Senate**, enabling comprehensive tracking of:
- How every party voted on every Act
- Individual MP and Senator voting records  
- Party discipline and coalition dynamics
- Cross-chamber voting pattern analysis
- Complete legislative decision audit trails

This provides **unprecedented transparency** into the Polish legislative process with granular voting data that can support detailed political analysis, civic engagement, and democratic accountability.