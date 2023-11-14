.. dbctl documentation master file, created by
   sphinx-quickstart on Sun Sep 24 21:08:46 2023.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

dbctl's documentation!
=================================

DBCTL is a tool designed to make running a database in a Docker container easy and fast. It offers the following features:

- Run a database with applied migration files and sample data.
- Access the connection URL and use your preferred tools to connect to the database.
- Launch a user interface (UI) for managing the database, if available.
- Enable users to quickly run tests in a fresh database.
- Cleanup databases once you're finished using them!

.. warning:: DBCTL is not intended for running databases in production. Its primary purpose is to simplify testing and practice with no hassle.

Todo
------------

- Setup and run MongoDB
- Support lua lang for redis in fixtures and migration scripts
- Support for js in fixtures and migration scripts
- Utilize golang templates to generate sample data.
- Python client (in progress)`PR <https://github.com/mirzakhany/dbctl/pull/17>`_
- JS client


Contributing
------------
We welcome any and all contributions! Here are some ways you can get started:
1. Report bugs: If you encounter any bugs, please let us know. Open up an issue and let us know the problem.
2. Contribute code: If you are a developer and want to contribute, follow the instructions below to get started!
3. Suggestions: If you don't want to code but have some awesome ideas, open up an issue explaining some updates or improvements you would like to see!
4. Documentation: If you see the need for some additional documentation, feel free to add some!

Instructions
************
1. Fork this repository
2. Clone the forked repository
3. Add your contributions (code or documentation)
4. Commit and push
5. Wait for pull request to be merged


.. toctree::
   :maxdepth: 2
   :caption: Overview
   :hidden:

   ./overview/install.md


.. toctree::
   :maxdepth: 2
   :caption: Getting Started
   :hidden:

   getting-started/postgres.md
   getting-started/redis.md

.. toctree::
   :maxdepth: 2
   :caption: Testing with DBCTL
   :hidden:

   testing/overview.md
   testing/golang.md

.. toctree::
   :maxdepth: 2
   :caption: Reference
   :hidden:

   reference/cli.md
   reference/manage.md
