# Assure

Assure is a system testing framework aimed at improving software testability. It provides a rich and flexible framework that is used to implement system level tests for any system.

Assure does not aim to know anything about the system under test (SUT) and as such does not make any assumptions not have any preferences around technologies or protocols. Assure does, however, have a strict view on how a system test should be monitored, see the [monitoring](#Monitoring) section for more information. 

The user is tasked with implementing testing modules to support performing operations against the SUT, which are then compiled together with the Assure framework to produce an executable that is used for testing. The Assure framework provides utilities and traffic generation, the user provides concrete implementation for how to perform interactions with the SUT.

## Writing testing modules

Testing modules in Assure are concrete implementations of how to perform operations towards a SUT. For example, an Assure testing module for a REST API would probably include a bunch of different HTTP requests. A testing module can be written either as a simple module, meaning Assure does not know anything about how it interacts with the underlying SUT, or as a verbose module which exposes a set list of possible operations.

Modules are registerd with the Assure manager, the root level component of the framework, which then makes them available in the executable. It is possible to enable/disable modules and tweak any settings that the testing modules expose through command line arguments. Command line arguments are automatically added for each module that is registered with the Assure manager.

### Simple testing modules

A testing module that does not expose any operations is considered a simple module. A simple module must itself implement traffic generation, as Assure is not informed of any specific operations.

### Verbose testing modules

Verbose testing modules expose what operations it supports. The exposed operations are made available as configuration options in the executable, where the user is able to determine frequencies (calls per minute per operation) and (optionally) any input arguments. Verbose modules provide the user the freedom of specifying settings per operation when running their tests.

## Reproducability with traffic models

In order to easily reproduce tests between software releases and provide support for a GitOps way of working, Assure provides a traffic model feature which lets the user write test files that can be given to the executable. Traffic models are YAML files that specify how Assure should behave when a test is started. The YAML file specifies which testing modules should be enabled and disabled, and settings for different operations.

Generate a blank traffic model using the executable using the `--generate-traffic-model` argument.

## Monitoring

Assure approaches system monitoring in a black box manner. The following monitoring methods are available with Assure's built in monitoring tools:

 1. CPU
 2. Memory
 3. (Prometheus) metrics
 4. Structured (JSON) logs

More monitoring tools may be added in the future, but it is not something that we do on a whim. We believe that a well designed system should be perfectly possible to monitor using the above methods. Adding any sort of white box monitoring is out of the question and because of this it is not possible to implement custom modules for monitoring. We believe such tests should be done on a lower level, not on and integration/system level.

### Thresholds

Thresholds can be set for either monitoring option to determine a level (for CPU, memory, one or more metric(s), or logs) where a limit is passed that warrants some kind of notice/warning/error. 

## Reporting

Assure has a reporting option which generates a YAML report on completion. The YAML file summarizes the test result by stating:

 - (Optional) Test name
 - (Optional) SUT version
 - Start datetime
 - Duration
 - Traffic model
 - Monitoring
   - CPU
     - Average
     - High
     - Low
   - Memory
     - Average
     - High
     - Low
   - Notices
   - Warnings
   - Errors
