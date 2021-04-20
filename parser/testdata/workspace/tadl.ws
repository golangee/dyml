// top level domain describes the companies main domain.
# ...is a software development and consulting company.
# The company develops various apps for different customers
# and some of them have special requirements to support
# their end-customers.
# see requirements::define::me1
# see requirements::define::me2
# see requirements::define::me3
# see requirements::define::me4
# see requirements::define::me5
# see requirements::define::me6
domain worldiety {

        # ...is a part of the company to support custom applications and their users.
        # see requirements::tickets::ManageTickets
        subdomain ApplicationSupport {

            # ...is one part of the application support.
            subdomain ErrorTracking {

                # ...describes crash incidents from the end users perspective.
                # By definition it has its own vocabulary, which means, that each
                # identifier must be unique and exactly defined.
                # see requirements::tickets::ManageTickets
                context Ticket

                # ...treats small and very large files submitted by an end user.
                context File

                # ...represents the identity and access management rules for the enclosing subdomain.
                context IAM
            }
        }

        # ...is not yet modelled.
        subdomain Development

        # ...is not yet modelled.
        subdomain Invoicing

        # ...this IAM has (mostly) nothing to do with an IAM context of a subdomain.
        subdomain IAM
}