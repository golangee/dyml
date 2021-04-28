As an YASA employee, I want unprocessed tickets to be automatically deleted after a period of time, so that I don't accidentally accumulate app user information that is highly unlikely to be relevant.

== Scenario: DoubleDelete
Given I'm a SupportietyAdmin
when I delete the same ticket twice
then I want a message telling me that its not possible.

== Scenario: DoubleDelete2
Given I'm a SupportietyAdmin
when I delete the same ticket twice
then I want a message telling me that its not possible.
