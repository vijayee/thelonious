contract.storage[10] = 12
return compile {
    contract.storage[5] = this.data[0]
}   
