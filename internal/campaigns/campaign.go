package campaigns

import "time"

// CampaignState defines the possible states of a Campaign
type CampaignState string

const (
	CampaignStateAny    CampaignState = "ANY"
	CampaignStateOpen   CampaignState = "OPEN"
	CampaignStateClosed CampaignState = "CLOSED"
)

// A Campaign of changesets over multiple Repos over time.
type Campaign struct {
	ID          int64
	Name        string
	Description string

	CampaignSpecID int64

	InitialApplierID int32
	LastApplierID    int32
	LastAppliedAt    time.Time

	NamespaceUserID int32
	NamespaceOrgID  int32

	ChangesetIDs []int64

	ClosedAt time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Clone returns a clone of a Campaign.
func (c *Campaign) Clone() *Campaign {
	cc := *c
	cc.ChangesetIDs = c.ChangesetIDs[:len(c.ChangesetIDs):len(c.ChangesetIDs)]
	return &cc
}

// RemoveChangesetID removes the given id from the Campaigns ChangesetIDs slice.
// If the id is not in ChangesetIDs calling this method doesn't have an effect.
func (c *Campaign) RemoveChangesetID(id int64) {
	for i := len(c.ChangesetIDs) - 1; i >= 0; i-- {
		if c.ChangesetIDs[i] == id {
			c.ChangesetIDs = append(c.ChangesetIDs[:i], c.ChangesetIDs[i+1:]...)
		}
	}
}

// Closed returns true when the ClosedAt timestamp has been set.
func (c *Campaign) Closed() bool { return !c.ClosedAt.IsZero() }
