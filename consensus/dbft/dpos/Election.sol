pragma solidity ^0.4.24;

contract Election {
    using SafeMath for uint;

    uint constant EPOCH_LENGH = 30000;
    uint constant MIN_DEPOSIT = 1000000 * (10 ** 18);
    bool constant PREV = false;
    bool constant NEXT = true;
    uint constant DEPOSIT_PERIOD = 1000000;

    struct VoteInfo {
        uint balance;
        uint number; // block number
    }


    uint public maxValidators = 5;
    //candidata => balance
    mapping(address => uint) public candidates;
    mapping(address => mapping(bool => address)) candidatesList;
    address listHead;

    // voter => candidate => vote infomation
    mapping(address => mapping(address => VoteInfo)) public voters;


    event Vote(address voter, address candidate, uint balance);
    event Withdraw(address voter, address candidate, uint balance);


    function vote(address candidate) public payable {
        require(msg.value >= MIN_DEPOSIT);

        if (isCandidate(candidate))  {
            candidates[candidate] = candidates[candidate].add(msg.value);
            promoteCandidate(candidate);
        } else {
            candidates[candidate] = msg.value;
            insertCandidate(candidate);
        }

        recordVote(msg.sender, candidate, msg.value);

        emit Vote(msg.sender, candidate, msg.value);
    }

    function withdraw(address candidate) public returns (bool) {
        require(voters[msg.sender][candidate].balance > 0);
        require(block.number > voters[msg.sender][candidate].number + DEPOSIT_PERIOD);

        uint balance = voters[msg.sender][candidate].balance;
        voters[msg.sender][candidate].balance = 0;
        voters[msg.sender][candidate].number = 0;

        msg.sender.transfer(balance);

        candidates[candidate] = candidates[candidate].sub(balance);
        demoteCandidate(candidate);

        emit Withdraw(msg.sender, candidate, balance);
    }

    function getValidators() public view returns (address[]) {
        address[] memory validators = new address[](validatorsCount());
        uint i = 0;
        for (address index = listHead; index != address(0); index = candidatesList[index][NEXT]) {
            validators[i] = index;
            i++;
            if (i >= maxValidators) {
                return validators;
            }
        }

        return validators;
    }

    function validatorsCount() public view returns (uint count) {
        for (address index = listHead; index != address(0); index = candidatesList[index][NEXT]) {
            count++;
        }

        return count;
    }

    function isCandidate(address candidate) public view returns (bool) {
        return candidates[candidate] > 0;
    }

    function recordVote(address voter, address candidate, uint balance) private {
        voters[voter][candidate].balance = voters[voter][candidate].balance.add(balance);
        voters[voter][candidate].number = block.number;
    }

    function insertCandidate(address candidate) private {
        if (listHead == address(0)) {
            listHead = candidate;
            return;
        }

        for (address index = listHead; ; index = candidatesList[index][NEXT]) {
            if (candidates[candidate] > candidates[index]) {
                insertCandidateBefore(candidate, index);
                return;
            }

            if (candidatesList[index][NEXT] == address(0)) {
                break;
            }
        }

        insertCandidateAfter(candidate, index);
    }

    function promoteCandidate(address candidate) private {
        address prev = candidatesList[candidate][PREV];
        if (prev == address(0) || candidates[prev] >= candidates[candidate]) {
            return;
        }

        popCandidate(candidate);

        for (prev = candidatesList[candidate][PREV]; prev != address(0); prev = candidatesList[prev][PREV]) {
            if (candidates[prev] > candidates[candidate]) {
                insertCandidateAfter(candidate, prev);
                return;
            }
        }

        insertCandidateBefore(candidate, listHead);
    }

    function demoteCandidate(address candidate) private {
        address next = candidatesList[candidate][NEXT];
        if (next == address(0) || candidates[next] < candidates[candidate]) {
            return;
        }

        popCandidate(candidate);
        if (candidates[candidate] == 0) {
            return;
        }
        for (next = candidatesList[candidate][NEXT]; ; next = candidatesList[next][NEXT]) {
            if (candidates[next] < candidates[candidate]) {
                insertCandidateBefore(candidate, next);
                return;
            }

            if (candidatesList[next][NEXT] == address(0)) {
                break;
            }
        }

        insertCandidateAfter(candidate, next);
    }

    function popCandidate(address candidate) private {
        if (listHead == candidate) {
            listHead = candidatesList[candidate][NEXT];
        }
        createLink(candidatesList[candidate][PREV], candidatesList[candidate][NEXT]);
        candidatesList[candidate][PREV] = address(0);
        candidatesList[candidate][NEXT] = address(0);
    }

    function insertCandidateBefore(address candidate, address to) private {
        createLink(candidatesList[to][PREV], candidate);
        if (to == listHead) {
            listHead = candidate;
        }
        createLink(candidate, to);
    }

    function insertCandidateAfter(address candidate, address to) private {
        createLink(candidate, candidatesList[to][NEXT]);
        createLink(to, candidate);
    }

    function createLink(address prev, address next) private {
        if (prev != address(0)) {
            candidatesList[prev][NEXT] = next;
        }
        if (next != address(0)) {
            candidatesList[next][PREV] = prev;
        }
    }
}


library SafeMath {
    /**
    * @dev Multiplies two numbers, throws on overflow.
    */
    function mul(uint256 a, uint256 b) internal pure returns (uint256) {
        if (a == 0) {
            return 0;
        }
        uint256 c = a * b;
        assert(c / a == b);
        return c;
    }

    /**
    * @dev Integer division of two numbers, truncating the quotient.
    */
    function div(uint256 a, uint256 b) internal pure returns (uint256) {
        // assert(b > 0); // Solidity automatically throws when dividing by 0
        uint256 c = a / b;
        // assert(a == b * c + a % b); // There is no case in which this doesn't hold
        return c;
    }

    /**
    * @dev Subtracts two numbers, throws on overflow (i.e. if subtrahend is greater than minuend).
    */
    function sub(uint256 a, uint256 b) internal pure returns (uint256) {
        assert(b <= a);
        return a - b;
    }

    /**
    * @dev Adds two numbers, throws on overflow.
    */
    function add(uint256 a, uint256 b) internal pure returns (uint256) {
        uint256 c = a + b;
        assert(c >= a);
        return c;
    }
}

//contract Vote{
//    address[] candidates;
//    uint public test;
//
//    constructor() public {}
//    function () public payable{}
//
//    function getCandidates() public view returns (address[]) {
//        return candidates;
//    }
//}

/*
"storage": {
"0x0000000000000000000000000000000000000000000000000000000000000001": "0x02",
"0x0000000000000000000000000000000000000000000000000000000000000000": "0x04", 数组长度

 web3.sha3("0x0000000000000000000000000000000000000000000000000000000000000000", {"encoding": "hex"}) = "0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563"
"0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563": "0xd8Dba507e85F116b1f7e231cA8525fC9008A6966",
"0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e564": "0x6571D97f340c8495B661a823F2C2145cA47D63c2",
"0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e565": "0xe36cbeB565B061217930767886474e3cDe903AC5",
"0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e566": "0xF512a992F3fb749857d758fFDa1330e590fa915E"
},
*/