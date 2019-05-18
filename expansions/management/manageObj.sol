pragma solidity ^0.4.24;

contract manageObj {
    mapping(address => bool) managers;
    mapping(address => bool) whitelist;

    function setWhiteList(address tokenid) public;
    function delWhiteList(address tokenid) public;
}
