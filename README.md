# Terraform Provider arc 

## Whats different?

Aws accounts with terraform do not flow well. It takes 90 days to delete an account. And usually when debugging/managing infra you will want quick delete/creates to test. 

This provider fixes that, by instead of deleting the account. It is moved to a "closed" folder. That way the account can either be manually closed by the user. Or quickly swapped between a closed folder and actual usage. 

If an account with the same name is closed, this will fail. Creating a better management for aws accounts and terraform resource management. 
