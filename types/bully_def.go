package types

const (
	BullyElection    = iota // Sent to announce election
	BullyAnswer             // Responds to the Election message
	BullyCoordinator        // Sent by winner of the election to announce victory
)

// 	BullyMessage is used to implement the Bully algorithm
// 	The algorithm uses the following message types:
// 		BullyElection: Sent to announce election.
//		BullyAnswer: Responds to the Election message.
//		BullyCoordinator: Sent by winner of the election to announce victory.
//
//  If P has the highest process ID, it sends a Victory message to all other processes and becomes the new Coordinator.
// 		Otherwise, P broadcasts an Election message to all other processes with higher process IDs than itself.
//	If P receives no Answer after sending an Election message,
//		then it broadcasts a Victory message to all other processes and becomes the Coordinator.
//	If P receives an Answer from a process with a higher ID,
//		it sends no further messages for this election and waits for a Victory message.
//		(If there is no Victory message after a period of time, it restarts the process at the beginning.)
//	If P receives an Election message from another process with a lower ID
//		it sends an Answer message back and starts the election process at the beginning, by sending an Election message to higher-numbered processes.
//	If P receives a Coordinator message,
//		it treats the sender as the coordinator.
type BullyMessage struct {
	PaxosID uint
	Type    int // uses the type defined above
}
