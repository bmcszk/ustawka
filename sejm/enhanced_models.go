//revive:disable:max-public-structs
package sejm

import (
	"context"
	"time"
)

// EnhancedAct represents a legislative act with comprehensive lifecycle information
type EnhancedAct struct {
	// Existing Act fields
	ID        string `json:"ELI"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Published string `json:"promulgation"`
	Position  int    `json:"pos"`
	Year      int    `json:"year"`
	Type      string `json:"type"`
	Address   string `json:"address"`

	// Enhanced lifecycle fields
	DetailedStatus     string    `json:"detailed_status"`
	CurrentStage       string    `json:"current_stage"`
	StageDate          time.Time `json:"stage_date"`
	DaysInStage        int       `json:"days_in_stage"`
	InitiatorType      string    `json:"initiator_type"`
	CommitteeCode      string    `json:"committee_code"`
	RapporteurName     string    `json:"rapporteur_name"`
	UrgencyStatus      string    `json:"urgency_status"`
	EUCompliance       bool      `json:"eu_compliance"`
	ProcessPrintNumber string    `json:"process_print_number"`
	RCLLink           string    `json:"rcl_link"`

	// Voting information
	SejmVotes       []VotingRecord `json:"sejm_votes"`
	SenateVotes     []VotingRecord `json:"senate_votes"`
	PartyBreakdowns map[string]PartyVote `json:"party_breakdowns"`

	// Lifecycle tracking
	Stages []ProcessStage `json:"stages"`

	// Metadata
	Tags  []string `json:"tags"`
	Links ActLinks `json:"links"`
}

// VotingRecord represents a voting event for an Act
type VotingRecord struct {
	ID               int                      `json:"id"`
	Date             time.Time                `json:"date"`
	VoteType         string                   `json:"vote_type"` // first_reading, amendment, final_passage, override
	ProceedingNumber int                      `json:"proceeding_number"`
	VotingNumber     int                      `json:"voting_number"`
	Result           string                   `json:"result"` // passed, failed
	TotalVoted       int                      `json:"total_voted"`
	YesVotes         int                      `json:"yes_votes"`
	NoVotes          int                      `json:"no_votes"`
	AbstainVotes     int                      `json:"abstain_votes"`
	AbsentVotes      int                      `json:"absent_votes"`
	PartyBreakdown   map[string]PartyVote     `json:"party_breakdown"`
	IndividualVotes  []IndividualVote         `json:"individual_votes"`
}

// PartyVote represents voting breakdown by political party
type PartyVote struct {
	Party         string  `json:"party"`
	PartyCode     string  `json:"party_code"`
	TotalMembers  int     `json:"total_members"`
	YesVotes      int     `json:"yes_votes"`
	NoVotes       int     `json:"no_votes"`
	AbstainVotes  int     `json:"abstain_votes"`
	AbsentVotes   int     `json:"absent_votes"`
	DisciplineRate float64 `json:"discipline_rate"` // Percentage voting with party majority
}

// IndividualVote represents a single MP or Senator vote
type IndividualVote struct {
	MemberID   int    `json:"member_id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Party      string `json:"party"`
	PartyCode  string `json:"party_code"`
	Vote       string `json:"vote"` // YES, NO, ABSTAIN, ABSENT
}

// ProcessStage represents a stage in the legislative process
type ProcessStage struct {
	ID            int       `json:"id"`
	StageName     string    `json:"stage_name"`
	StageDate     time.Time `json:"stage_date"`
	StageOrder    int       `json:"stage_order"`
	CommitteeCode string    `json:"committee_code"`
	CommitteeName string    `json:"committee_name"`
	RapporteurName string   `json:"rapporteur_name"`
	Notes         string    `json:"notes"`
	IsCurrent     bool      `json:"is_current"`
	DurationDays  int       `json:"duration_days"`
	PrintNumbers  []string  `json:"print_numbers"`
}

// ActLinks represents external links for an Act
type ActLinks struct {
	PDFDocument   string `json:"pdf_document"`
	SejmProcess   string `json:"sejm_process"`
	VotingDetails string `json:"voting_details"`
	RCLPortal     string `json:"rcl_portal"`
}

// ProcessInfo represents information from Sejm process tracking API
type ProcessInfo struct {
	Number              string         `json:"number"`
	Title               string         `json:"title"`
	DocumentType        string         `json:"documentType"`
	DocumentDate        string         `json:"documentDate"`
	ProcessStartDate    string         `json:"processStartDate"`
	UrgencyStatus       string         `json:"urgencyStatus"`
	PrincipleOfSubsidiarity bool       `json:"principleOfSubsidiarity"`
	LegislativeCommittee bool          `json:"legislativeCommittee"`
	Passed              bool           `json:"passed"`
	Stages              []ProcessStageAPI `json:"stages"`
}

// ProcessStageAPI represents a stage from the Sejm API
type ProcessStageAPI struct {
	StageName   string                `json:"stageName"`
	Date        string                `json:"date"`
	PrintNumber string                `json:"printNumber"`
	Children    []ProcessStageAPI     `json:"children"`
}

// VotingInfo represents voting information from Sejm API
type VotingInfo struct {
	Description   string       `json:"description"`
	Title         string       `json:"title"`
	Topic         string       `json:"topic"`
	TotalVoted    int          `json:"totalVoted"`
	Yes           int          `json:"yes"`
	No            int          `json:"no"`
	Abstain       int          `json:"abstain"`
	MajorityVotes int          `json:"majorityVotes"`
	MajorityType  string       `json:"majorityType"`
	Votes         []MPVote     `json:"votes"`
}

// MPVote represents an individual MP vote from Sejm API
type MPVote struct {
	MP        int    `json:"MP"`
	Club      string `json:"club"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Vote      string `json:"vote"`
}

// SenateVotingInfo represents Senate voting data
type SenateVotingInfo struct {
	SessionNumber int                        `json:"session_number"`
	VoteNumber    int                        `json:"vote_number"`
	Date          time.Time                  `json:"date"`
	ActTitle      string                     `json:"act_title"`
	VoteType      string                     `json:"vote_type"` // accept, amend, reject
	TotalVoted    int                        `json:"total_voted"`
	PartyResults  map[string]SenatePartyVote `json:"party_results"`
}

// SenatePartyVote represents Senate party voting breakdown
type SenatePartyVote struct {
	ClubName      string `json:"club_name"`
	TotalMembers  int    `json:"total_members"`
	Voted         int    `json:"voted"`
	YesVotes      int    `json:"yes_votes"`
	NoVotes       int    `json:"no_votes"`
	AbstainVotes  int    `json:"abstain_votes"`
	NotVoted      int    `json:"not_voted"`
}

// StatusMapping defines the mapping between API statuses and enhanced statuses
var StatusMapping = map[string]string{
	// Submission and initial stages
	"Projekt wpłynął do Sejmu":                    "submitted",
	"Skierowano do I czytania w komisjach":        "committee_first_reading",
	"I czytanie w komisjach":                      "committee_first_reading",
	"Praca w komisjach po I czytaniu":             "committee_work",
	
	// Parliamentary readings
	"II czytanie":                                 "second_reading",
	"III czytanie":                               "third_reading",
	"Ustawa przeszła przez Sejm":                  "passed_sejm",
	
	// Senate process
	"Przekazano do Senatu":                        "senate_review",
	"Senat przyjął bez poprawek":                  "senate_accepted",
	"Senat przyjął z poprawkami":                  "senate_amended",
	"Senat odrzucił":                             "senate_rejected",
	"Sejm odrzucił poprawki Senatu":              "override_vote",
	
	// Presidential stage
	"Przekazano do Prezydenta":                    "presidential_review",
	"Prezydent podpisał":                         "presidential_signed",
	"Prezydent zawetował":                        "presidential_veto",
	"Sejm odrzucił weto":                         "veto_override",
	
	// Publication
	"Opublikowano w Dzienniku Ustaw":             "published",
	"Ustawa weszła w życie":                      "in_force",
	
	// Special statuses
	"Tryb pilny":                                 "urgent",
	"Implementacja prawa UE":                     "eu_compliance",
}

// GetEnhancedStatus converts API status to enhanced status
func GetEnhancedStatus(apiStatus string) string {
	if enhanced, ok := StatusMapping[apiStatus]; ok {
		return enhanced
	}
	return "unknown"
}

// CalculateDaysInStage calculates days spent in current stage
func CalculateDaysInStage(stageDate time.Time) int {
	if stageDate.IsZero() {
		return 0
	}
	return int(time.Since(stageDate).Hours() / 24)
}

// DetermineInitiatorType determines the type of initiator based on process info
func DetermineInitiatorType(processInfo *ProcessInfo) string {
	// This would be enhanced based on actual API data patterns
	switch processInfo.DocumentType {
	case "projekt ustawy rządowy":
		return "government"
	case "projekt ustawy poselski":
		return "deputy"
	case "projekt ustawy senacki":
		return "senate"
	case "projekt ustawy obywatelski":
		return "citizen"
	default:
		return "unknown"
	}
}

// GenerateTags generates tags based on Act properties
func GenerateTags(act *EnhancedAct, processInfo *ProcessInfo) []string {
	var tags []string
	
	if act.UrgencyStatus == "urgent" || processInfo.UrgencyStatus == "URGENT" {
		tags = append(tags, "urgent")
	}
	
	if act.EUCompliance || processInfo.PrincipleOfSubsidiarity {
		tags = append(tags, "eu-law")
	}
	
	if processInfo.LegislativeCommittee {
		tags = append(tags, "legislative-committee")
	}
	
	// Add type-based tags
	switch act.Type {
	case "ustawa":
		tags = append(tags, "act")
	case "rozporządzenie":
		tags = append(tags, "regulation")
	case "uchwała":
		tags = append(tags, "resolution")
	}
	
	return tags
}

// GenerateActLinks generates relevant links for an Act
func GenerateActLinks(act *EnhancedAct) ActLinks {
	links := ActLinks{}
	
	if act.Address != "" {
		links.PDFDocument = act.Address
	}
	
	if act.ProcessPrintNumber != "" {
		// Generate Sejm process link
		links.SejmProcess = "https://www.sejm.gov.pl/sejm10.nsf/druk.xsp?nr=" + act.ProcessPrintNumber
	}
	
	if act.RCLLink != "" {
		links.RCLPortal = act.RCLLink
	}
	
	return links
}

// ActLinkingService handles linking between Sejm and Senate data
type ActLinkingService struct {
	sejmClient   *Client
	senateClient SenateClient
}

// NewActLinkingService creates a new ActLinkingService
func NewActLinkingService(sejmClient *Client, senateClient SenateClient) *ActLinkingService {
	return &ActLinkingService{
		sejmClient:   sejmClient,
		senateClient: senateClient,
	}
}

// LinkSenateToSejm links Senate voting data to Sejm Acts
func (als *ActLinkingService) LinkSenateToSejm(_ context.Context, sejmAct *EnhancedAct,
	senateVotes []SenateVotingRecord) error {
	// This would implement the logic to match Senate votes to Sejm Acts
	// For now, we'll use a simple title matching approach
	
	for _, vote := range senateVotes {
		if als.matchActToVote(sejmAct, vote) {
			// Convert SenateVotingRecord to VotingRecord format
			votingRecord := VotingRecord{
				Date:         vote.VotingDate,
				VoteType:     "senate_review",
				Result:       vote.Result,
				YesVotes:     vote.VotesFor,
				NoVotes:      vote.VotesAgainst,
				AbstainVotes: vote.VotesAbstain,
				TotalVoted:   vote.VotesFor + vote.VotesAgainst + vote.VotesAbstain,
			}
			
			sejmAct.SenateVotes = append(sejmAct.SenateVotes, votingRecord)
		}
	}
	
	return nil
}

// matchActToVote determines if a Senate vote matches a Sejm Act
func (*ActLinkingService) matchActToVote(act *EnhancedAct, vote SenateVotingRecord) bool {
	// Simple title matching - could be enhanced with more sophisticated matching
	return act.Title == vote.Subject || act.ID == vote.ActID
}