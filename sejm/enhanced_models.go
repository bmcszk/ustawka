//revive:disable:max-public-structs
package sejm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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

// GetYearString returns the year as a string for template rendering
func (e *EnhancedAct) GetYearString() string {
	if e.Year == 0 {
		return ""
	}
	return fmt.Sprintf("%d", e.Year)
}

// ParliamentaryProcess represents an active legislative process from the Sejm API
type ParliamentaryProcess struct {
	Number                  string                     `json:"number"`
	Title                   string                     `json:"title"`
	TitleFinal              string                     `json:"titleFinal,omitempty"`
	DocumentType            string                     `json:"documentType"`
	DocumentDate            string                     `json:"documentDate"`
	ProcessStartDate        string                     `json:"processStartDate"`
	ChangeDate              string                     `json:"changeDate"`
	ClosureDate             string                     `json:"closureDate,omitempty"`
	Passed                  bool                       `json:"passed"`
	ELI                     string                     `json:"ELI,omitempty"`
	Address                 string                     `json:"address,omitempty"`
	DisplayAddress          string                     `json:"displayAddress,omitempty"`
	Term                    int                        `json:"term"`
	UrgencyStatus           string                     `json:"urgencyStatus"`
	ShortenProcedure        bool                       `json:"shortenProcedure"`
	LegislativeCommittee    bool                       `json:"legislativeCommittee"`
	PrincipleOfSubsidiarity bool                       `json:"principleOfSubsidiarity"`
	UE                      string                     `json:"UE"`
	Comments                string                     `json:"comments,omitempty"`
	Description             string                     `json:"description,omitempty"`
	Stages                  []ParliamentaryStage       `json:"stages"`
	PrintsConsideredJointly []string                   `json:"printsConsideredJointly,omitempty"`
	Links                   []ParliamentaryLink        `json:"links,omitempty"`
	WebGeneratedDate        string                     `json:"webGeneratedDate"`
}

// ParliamentaryStage represents a stage in the legislative process
type ParliamentaryStage struct {
	Date         string                      `json:"date"`
	StageName    string                      `json:"stageName"`
	PrintNumber  string                      `json:"printNumber,omitempty"`
	SittingNum   int                         `json:"sittingNum,omitempty"`
	Children     []ParliamentaryStageChild   `json:"children,omitempty"`
}

// ParliamentaryStageChild represents a sub-stage (like committee referral)
type ParliamentaryStageChild struct {
	Date           string `json:"date"`
	StageName      string `json:"stageName"`
	CommitteeCode  string `json:"committeeCode,omitempty"`
	Type           string `json:"type"`
}

// ParliamentaryLink represents a link to external resources
type ParliamentaryLink struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

// ParliamentaryStageMapping maps parliamentary stage names to Kanban column names
var ParliamentaryStageMapping = map[string]string{
	// Initial submission
	"Projekt wpłynął do Sejmu":                           "submitted",
	"Projekt wpłynął do Senatu":                          "submitted",
	
	// Committee work
	"Skierowanie do komisji":                             "committee_work",
	"Skierowano do komisji":                              "committee_work", 
	"Komisja zakończyła prace":                           "committee_work",
	"Posiedzenie komisji":                                "committee_work",
	
	// Sejm readings
	"Skierowano do I czytania na posiedzeniu Sejmu":      "sejm_readings",
	"I czytanie na posiedzeniu Sejmu":                    "sejm_readings",
	"II czytanie na posiedzeniu Sejmu":                   "sejm_readings",
	"III czytanie na posiedzeniu Sejmu":                  "sejm_readings",
	"Przegłosowanie w Sejmie":                            "sejm_readings",
	"Uchwalono":                                          "sejm_readings",
	
	// Senate review
	"Przekazano do Senatu":                               "senate_review",
	"Wpłynął do Senatu":                                  "senate_review",
	"Posiedzenie Senatu":                                 "senate_review",
	"Senat nie wniósł poprawek":                          "senate_review",
	"Senat wniósł poprawki":                              "senate_review",
	"Senat odrzucił ustawę":                              "senate_review",
	
	// Presidential review
	"Przekazano do Prezydenta":                           "presidential_review",
	"Prezydent podpisał":                                 "presidential_review",
	"Prezydent zawetował":                                "presidential_review",
	"Odrzucenie weta":                                    "presidential_review",
	
	// Publication and completion
	"Opublikowano w Dzienniku Ustaw":                     "published",
	"Ustawa weszła w życie":                              "in_force",
	"Uchwalono uchwałę":                                  "published",
}

// ConvertParliamentaryProcessToEnhancedAct converts parliamentary process to EnhancedAct
func ConvertParliamentaryProcessToEnhancedAct(process *ParliamentaryProcess) *EnhancedAct {
	// Determine current stage and status
	currentStage, detailedStatus := determineProcessStage(process)
	
	// Parse year from process number or document date
	year := extractYearFromProcess(process)
	
	enhanced := &EnhancedAct{
		ID:                  process.ELI,
		Title:               process.Title,
		Status:              mapProcessStatusToBasic(process),
		Published:           process.DocumentDate,
		Position:            parsePositionFromNumber(process.Number),
		Year:                year,
		Type:                mapDocumentTypeToType(process.DocumentType),
		Address:             process.Address,
		DetailedStatus:      detailedStatus,
		CurrentStage:        currentStage,
		StageDate:           parseLastStageDate(process),
		DaysInStage:         calculateDaysInCurrentStage(process),
		InitiatorType:       determineInitiatorFromDocumentType(process.DocumentType),
		CommitteeCode:       extractCommitteeFromStages(process.Stages),
		UrgencyStatus:       strings.ToLower(process.UrgencyStatus),
		EUCompliance:        process.PrincipleOfSubsidiarity || process.UE == "YES",
		ProcessPrintNumber:  process.Number,
		Stages:              convertParliamentaryStages(process.Stages),
		Tags:                generateParliamentaryTags(process),
		Links:               generateParliamentaryLinks(process),
	}
	
	return enhanced
}

// Helper functions for conversion
func determineProcessStage(process *ParliamentaryProcess) (stageName, detailedStatus string) {
	if len(process.Stages) == 0 {
		return "submitted", "submitted"
	}
	
	// Get the last stage
	lastStage := process.Stages[len(process.Stages)-1]
	stageName = lastStage.StageName
	
	// Map to detailed status
	if mappedStatus, ok := ParliamentaryStageMapping[stageName]; ok {
		return stageName, mappedStatus
	}
	
	// Default based on closure status
	if process.ClosureDate != "" {
		if process.Passed {
			return "Completed", "published"
		}
		return "Rejected", "rejected"
	}
	
	return stageName, "submitted"
}

func extractYearFromProcess(process *ParliamentaryProcess) int {
	// Try to extract from document date first
	if process.DocumentDate != "" {
		if date, err := time.Parse("2006-01-02", process.DocumentDate); err == nil {
			year := date.Year()
			slog.Debug("Extracted year from document date", "year", year, 
				"document_date", process.DocumentDate, "process", process.Number)
			return year
		}
	}
	
	// Fallback to current year
	currentYear := time.Now().Year()
	slog.Debug("Using current year for process", "year", currentYear, 
		"process", process.Number, "document_date", process.DocumentDate)
	return currentYear
}

func mapProcessStatusToBasic(process *ParliamentaryProcess) string {
	if process.ClosureDate != "" {
		if process.Passed {
			return "uchwalono"
		}
		return "odrzucono"
	}
	return "w toku"
}

func mapDocumentTypeToType(documentType string) string {
	switch documentType {
	case "projekt ustawy":
		return "ustawa"
	case "projekt uchwały":
		return "uchwała"
	case "projekt rozporządzenia":
		return "rozporządzenie"
	default:
		return documentType
	}
}

func parsePositionFromNumber(number string) int {
	if pos, err := fmt.Sscanf(number, "%d", new(int)); err == nil && pos == 1 {
		var result int
		if _, err := fmt.Sscanf(number, "%d", &result); err == nil {
			return result
		}
	}
	return 0
}

func parseLastStageDate(process *ParliamentaryProcess) time.Time {
	if len(process.Stages) == 0 {
		if process.ProcessStartDate != "" {
			if date, err := time.Parse("2006-01-02", process.ProcessStartDate); err == nil {
				return date
			}
		}
		return time.Time{}
	}
	
	lastStage := process.Stages[len(process.Stages)-1]
	if date, err := time.Parse("2006-01-02", lastStage.Date); err == nil {
		return date
	}
	
	return time.Time{}
}

func calculateDaysInCurrentStage(process *ParliamentaryProcess) int {
	stageDate := parseLastStageDate(process)
	if stageDate.IsZero() {
		return 0
	}
	return int(time.Since(stageDate).Hours() / 24)
}

func determineInitiatorFromDocumentType(documentType string) string {
	switch {
	case strings.Contains(strings.ToLower(documentType), "rządowy"):
		return "government"
	case strings.Contains(strings.ToLower(documentType), "poselski"):
		return "deputy"
	case strings.Contains(strings.ToLower(documentType), "senacki"):
		return "senate"
	case strings.Contains(strings.ToLower(documentType), "obywatelski"):
		return "citizen"
	default:
		return "unknown"
	}
}

func extractCommitteeFromStages(stages []ParliamentaryStage) string {
	for _, stage := range stages {
		for _, child := range stage.Children {
			if child.CommitteeCode != "" {
				return child.CommitteeCode
			}
		}
	}
	return ""
}

func convertParliamentaryStages(stages []ParliamentaryStage) []ProcessStage {
	var converted []ProcessStage
	
	for _, stage := range stages {
		stageDate, _ := time.Parse("2006-01-02", stage.Date)
		
		processStage := ProcessStage{
			StageName: stage.StageName,
			StageDate: stageDate,
			IsCurrent: false, // Will be determined later
		}
		
		// Add committee info if available
		for _, child := range stage.Children {
			if child.CommitteeCode != "" {
				processStage.CommitteeName = child.CommitteeCode
			}
		}
		
		converted = append(converted, processStage)
	}
	
	// Mark the last stage as current if process is ongoing
	if len(converted) > 0 {
		converted[len(converted)-1].IsCurrent = true
	}
	
	return converted
}

func generateParliamentaryTags(process *ParliamentaryProcess) []string {
	var tags []string
	
	if process.UrgencyStatus == "URGENT" {
		tags = append(tags, "urgent")
	}
	
	if process.PrincipleOfSubsidiarity || process.UE == "YES" {
		tags = append(tags, "eu-law")
	}
	
	if process.LegislativeCommittee {
		tags = append(tags, "legislative-committee")
	}
	
	// Add type-based tags
	switch process.DocumentType {
	case "projekt ustawy":
		tags = append(tags, "bill")
	case "projekt uchwały":
		tags = append(tags, "resolution")
	case "projekt rozporządzenia":
		tags = append(tags, "regulation")
	}
	
	// Add initiator tags
	if strings.Contains(strings.ToLower(process.DocumentType), "obywatelski") {
		tags = append(tags, "citizen-initiative")
	} else if strings.Contains(strings.ToLower(process.DocumentType), "rządowy") {
		tags = append(tags, "government-bill")
	}
	
	return tags
}

func generateParliamentaryLinks(process *ParliamentaryProcess) ActLinks {
	links := ActLinks{}
	
	for _, link := range process.Links {
		switch link.Rel {
		case "eli":
			links.RCLPortal = link.Href
		case "eli-api":
			// Could be used for additional API calls
		case "isap":
			links.PDFDocument = link.Href
		}
	}
	
	// Generate Sejm process link
	if process.Number != "" {
		links.SejmProcess = fmt.Sprintf("https://www.sejm.gov.pl/sejm%d.nsf/druk.xsp?nr=%s", 
			process.Term, process.Number)
	}
	
	return links
}