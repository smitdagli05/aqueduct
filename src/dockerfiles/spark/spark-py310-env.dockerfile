FROM quay.io/pypa/manylinux_2_24_x86_64:latest

SHELL [ "/bin/bash", "--login", "-c" ]

MAINTAINER Aqueduct <hello@aqueducthq.com> version: 0.0.1

USER root

RUN apt-get update && \
  apt-get install wget

# Install miniconda
ENV CONDA_DIR /opt/conda
RUN wget --quiet https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh -O ~/miniconda.sh && \
    /bin/bash ~/miniconda.sh -b -p /opt/conda

# Put conda in path so we can use conda activate
ENV PATH=$CONDA_DIR/bin:$PATH
RUN echo ". $CONDA_DIR/etc/profile.d/conda.sh" >> ~/.profile
RUN conda init bash
RUN conda create -n py310_env python=3.10
RUN echo "conda activate py310_env" >> ~/.bashrc

RUN	wget --quiet https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip -O awscliv2.zip && \
    unzip awscliv2.zip && ./aws/install 

RUN conda activate py310_env
RUN pip install conda-pack aqueduct-ml==0.2.9

COPY ./spark/create-conda-env.sh /

ENV PYTHONUNBUFFERED 1

CMD ["conda", "run", "-n", "py310_env", "/bin/bash", "/create-conda-env.sh"]