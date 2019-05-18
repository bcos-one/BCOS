pragma solidity ^0.4.24;

contract tokenObject {
    struct token {
        string name;
        uint256 supply;
        address manager;
        bool canIncrease;
        bool canBurn;
    }

    mapping(address => token) tokens;

    function issue(string name, address manager, address beneficiary, uint256 supply, bool canIncrease, bool canburn) public pure;
    function increase(address token, address beneficiary, uint256 amount) public pure;
    function burn(address token, uint256 amount) public pure;
}